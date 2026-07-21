package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func makeTestPaths(t *testing.T) (logPath string, statePath string, dbPath string) {
	t.Helper()
	root := t.TempDir()
	runtimeRoot := filepath.Join(root, "runtime")
	return filepath.Join(runtimeRoot, "echo.log"), filepath.Join(runtimeRoot, "state.json"), filepath.Join(runtimeRoot, "echo.sqlite")
}

func newTestHarnessApp(t *testing.T) *harnessApp {
	return newTestHarnessAppWithEnv(t, nil)
}

func newTestHarnessAppWithEnv(t *testing.T, extraEnv map[string]string) *harnessApp {
	t.Helper()

	logPath, statePath, dbPath := makeTestPaths(t)
	t.Setenv("ECHO_MESSAGE", "test message")
	t.Setenv("ECHO_LOG_PATH", logPath)
	t.Setenv("ECHO_STATE_PATH", statePath)
	t.Setenv("ECHO_DB_PATH", dbPath)
	for key, value := range extraEnv {
		t.Setenv(key, value)
	}

	app, err := newHarnessApp()
	if err != nil {
		t.Fatalf("newHarnessApp failed: %v", err)
	}

	t.Cleanup(func() {
		for _, child := range app.snapshot().Children {
			if child.ExitedAt == nil {
				if process, err := os.FindProcess(child.PID); err == nil {
					_ = process.Kill()
				}
			}
		}
		app.closeHealthServers()
		if app.httpServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = app.httpServer.Shutdown(ctx)
			cancel()
		}
		if app.logFile != nil {
			_ = app.logFile.Close()
		}
		if app.db != nil {
			_ = app.db.Close()
		}
	})

	return app
}

func postJSON(t *testing.T, handler http.HandlerFunc, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload failed: %v", err)
		}
		body = bytes.NewReader(raw)
	}

	request := httptest.NewRequest(http.MethodPost, "/", body)
	if payload != nil {
		request.Header.Set("content-type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

func getJSON(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	return recorder
}

func decodeJSONBody[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode JSON body failed: %v", err)
	}
	return body
}

func waitForCondition(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func findFreePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port failed: %v", err)
	}
	defer listener.Close()
	return fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)
}

func buildHarnessBinary(t *testing.T) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "echo-service-test")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}

	command := exec.Command("go", "build", "-o", output, ".")
	command.Dir = "."
	buildOutput, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(buildOutput))
	}

	return output
}

func startHarnessProcess(t *testing.T, binary string, env map[string]string) *exec.Cmd {
	t.Helper()

	command := exec.Command(binary)
	command.Dir = filepath.Dir(binary)
	command.Env = append(os.Environ(), "ECHO_MESSAGE=subprocess")
	for key, value := range env {
		command.Env = append(command.Env, key+"="+value)
	}

	if err := command.Start(); err != nil {
		t.Fatalf("start harness process failed: %v", err)
	}

	t.Cleanup(func() {
		if command.ProcessState == nil || !command.ProcessState.Exited() {
			_ = command.Process.Kill()
			_, _ = command.Process.Wait()
		}
	})

	return command
}

func waitForHTTPReady(t *testing.T, baseURL string) {
	t.Helper()
	waitForCondition(t, 10*time.Second, func() bool {
		response, err := http.Get(baseURL + "/health")
		if err != nil {
			return false
		}
		defer response.Body.Close()
		return response.StatusCode == http.StatusOK
	})
}

func waitForDedicatedHTTPStatus(t *testing.T, url string, expectedStatus int, expectedText string) {
	t.Helper()
	waitForCondition(t, 5*time.Second, func() bool {
		response, err := http.Get(url)
		if err != nil {
			return false
		}
		defer response.Body.Close()
		body, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return false
		}
		return response.StatusCode == expectedStatus && strings.Contains(string(body), expectedText)
	})
}

func waitForDedicatedHTTPUnavailable(t *testing.T, url string) {
	t.Helper()
	waitForCondition(t, 5*time.Second, func() bool {
		_, err := http.Get(url)
		return err != nil
	})
}

func waitForTCPPayload(t *testing.T, address string, expected string) {
	t.Helper()
	waitForCondition(t, 5*time.Second, func() bool {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			return false
		}
		defer connection.Close()
		_ = connection.SetDeadline(time.Now().Add(time.Second))
		body, readErr := io.ReadAll(connection)
		if readErr != nil {
			return false
		}
		return strings.Contains(string(body), expected)
	})
}

func waitForTCPUnavailable(t *testing.T, address string) {
	t.Helper()
	waitForCondition(t, 5*time.Second, func() bool {
		connection, err := net.DialTimeout("tcp", address, 250*time.Millisecond)
		if err == nil {
			connection.Close()
			return false
		}
		return true
	})
}

func TestNewHarnessAppPersistsStartupState(t *testing.T) {
	app := newTestHarnessApp(t)

	snapshot := app.snapshot()
	if snapshot.Service != "echo-service" {
		t.Fatalf("unexpected service name: %s", snapshot.Service)
	}
	if snapshot.Message != "test message" {
		t.Fatalf("unexpected message: %s", snapshot.Message)
	}
	if snapshot.ActionCount != 1 {
		t.Fatalf("expected startup event to count as one action, got %d", snapshot.ActionCount)
	}

	stateRaw, err := os.ReadFile(app.statePath)
	if err != nil {
		t.Fatalf("read state file failed: %v", err)
	}

	var persisted harnessSnapshot
	if err := json.Unmarshal(stateRaw, &persisted); err != nil {
		t.Fatalf("decode persisted state failed: %v", err)
	}
	if persisted.LastAction != "startup" {
		t.Fatalf("unexpected persisted lastAction: %s", persisted.LastAction)
	}

	var count int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count startup events failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one startup event row, got %d", count)
	}
}

func TestHandlersWriteLogsStateAndSQLite(t *testing.T) {
	app := newTestHarnessApp(t)

	writeLog := postJSON(t, app.handleWriteLog, actionRequest{Message: "hello log"})
	if writeLog.Code != http.StatusOK {
		t.Fatalf("write-log returned %d", writeLog.Code)
	}

	writeState := postJSON(t, app.handleWriteState, actionRequest{Message: "hello state"})
	if writeState.Code != http.StatusOK {
		t.Fatalf("write-state returned %d", writeState.Code)
	}

	writeSQLite := postJSON(t, app.handleWriteSQLite, actionRequest{Message: "hello sqlite"})
	if writeSQLite.Code != http.StatusOK {
		t.Fatalf("write-sqlite returned %d", writeSQLite.Code)
	}

	logs := getJSON(t, app.handleLogs)
	if logs.Code != http.StatusOK {
		t.Fatalf("logs returned %d", logs.Code)
	}
	var logsBody map[string]any
	if err := json.Unmarshal(logs.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs body failed: %v", err)
	}
	content := logsBody["content"].(string)
	for _, needle := range []string{"hello log", "hello state", "hello sqlite"} {
		if !strings.Contains(content, needle) {
			t.Fatalf("expected log output to contain %q", needle)
		}
	}

	state := getJSON(t, app.handleState)
	if state.Code != http.StatusOK {
		t.Fatalf("state returned %d", state.Code)
	}
	snapshot := decodeJSONBody[harnessSnapshot](t, state)
	if snapshot.LastAction != "write-sqlite" {
		t.Fatalf("unexpected last action: %s", snapshot.LastAction)
	}
	if snapshot.ActionCount < 4 {
		t.Fatalf("expected at least 4 actions including startup, got %d", snapshot.ActionCount)
	}

	sqliteView := getJSON(t, app.handleSQLite)
	if sqliteView.Code != http.StatusOK {
		t.Fatalf("sqlite returned %d", sqliteView.Code)
	}
	var sqliteBody struct {
		Events []eventRow `json:"events"`
	}
	if err := json.Unmarshal(sqliteView.Body.Bytes(), &sqliteBody); err != nil {
		t.Fatalf("decode sqlite body failed: %v", err)
	}
	if len(sqliteBody.Events) < 4 {
		t.Fatalf("expected at least 4 event rows, got %d", len(sqliteBody.Events))
	}
}

func TestErrorActionPersistsLastError(t *testing.T) {
	app := newTestHarnessApp(t)

	response := postJSON(t, app.handleErrorAction, actionRequest{Message: "boom"})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("error action returned %d", response.Code)
	}

	snapshot := app.snapshot()
	if snapshot.LastError != "boom" {
		t.Fatalf("unexpected lastError: %s", snapshot.LastError)
	}
	if snapshot.LastAction != "error" {
		t.Fatalf("unexpected lastAction: %s", snapshot.LastAction)
	}
}

func TestEnvAndServiceLassoOutputExposeGlobalEnv(t *testing.T) {
	app := newTestHarnessAppWithEnv(t, map[string]string{
		"SERVICE_LASSO_GLOBAL_ENV_JSON": `{"API_URL":"http://localhost:9000","TOKEN":"secret-token"}`,
		"SERVICE_LASSO_GLOBAL_REGION":   "apac",
		"GLOBAL_SHARED":                 "shared-value",
	})

	envResponse := getJSON(t, app.handleEnv)
	if envResponse.Code != http.StatusOK {
		t.Fatalf("env returned %d", envResponse.Code)
	}
	var envBody map[string]any
	if err := json.Unmarshal(envResponse.Body.Bytes(), &envBody); err != nil {
		t.Fatalf("decode env body failed: %v", err)
	}
	serviceEnv, ok := envBody["serviceEnv"].(map[string]any)
	if !ok {
		t.Fatalf("missing serviceEnv section")
	}
	if serviceEnv["ECHO_HTTP_HEALTH_PORT"] == "" || serviceEnv["ECHO_TCP_PORT"] == "" {
		t.Fatalf("expected health ports in serviceEnv")
	}

	globalResponse := getJSON(t, app.handleGlobalEnv)
	if globalResponse.Code != http.StatusOK {
		t.Fatalf("global-env returned %d", globalResponse.Code)
	}
	var globalBody map[string]any
	if err := json.Unmarshal(globalResponse.Body.Bytes(), &globalBody); err != nil {
		t.Fatalf("decode global-env body failed: %v", err)
	}

	globalEnv, ok := globalBody["globalEnv"].(map[string]any)
	if !ok {
		t.Fatalf("missing globalEnv section")
	}
	if globalEnv["API_URL"] != "http://localhost:9000" {
		t.Fatalf("unexpected API_URL value: %#v", globalEnv["API_URL"])
	}
	if globalEnv["REGION"] != "apac" {
		t.Fatalf("unexpected REGION value: %#v", globalEnv["REGION"])
	}
	if globalEnv["SHARED"] != "shared-value" {
		t.Fatalf("unexpected SHARED value: %#v", globalEnv["SHARED"])
	}

	serviceLassoResponse := getJSON(t, app.handleServiceLassoOutput)
	if serviceLassoResponse.Code != http.StatusOK {
		t.Fatalf("service-lasso/output returned %d", serviceLassoResponse.Code)
	}
	var serviceLassoBody map[string]any
	if err := json.Unmarshal(serviceLassoResponse.Body.Bytes(), &serviceLassoBody); err != nil {
		t.Fatalf("decode service-lasso/output body failed: %v", err)
	}
	if serviceLassoBody["serviceId"] != "echo-service" {
		t.Fatalf("unexpected serviceId: %#v", serviceLassoBody["serviceId"])
	}

	healthTargets, ok := serviceLassoBody["healthTargets"].(map[string]any)
	if !ok {
		t.Fatalf("missing healthTargets")
	}
	if _, ok := healthTargets["http"].(map[string]any); !ok {
		t.Fatalf("missing http health target")
	}
	if _, ok := healthTargets["tcp"].(map[string]any); !ok {
		t.Fatalf("missing tcp health target")
	}
}

func TestStdoutStderrAndHealthControls(t *testing.T) {
	httpHealthPort := findFreePort(t)
	tcpHealthPort := findFreePort(t)
	app := newTestHarnessAppWithEnv(t, map[string]string{
		"ECHO_HTTP_HEALTH_PORT": httpHealthPort,
		"ECHO_TCP_PORT":         tcpHealthPort,
	})

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	app.stdout = &stdoutBuffer
	app.stderr = &stderrBuffer

	stdoutResponse := postJSON(t, app.handleWriteStdout, actionRequest{Message: "stdout proof"})
	if stdoutResponse.Code != http.StatusOK {
		t.Fatalf("write-stdout returned %d", stdoutResponse.Code)
	}
	stderrResponse := postJSON(t, app.handleWriteStderr, actionRequest{Message: "stderr proof"})
	if stderrResponse.Code != http.StatusOK {
		t.Fatalf("write-stderr returned %d", stderrResponse.Code)
	}
	if !strings.Contains(stdoutBuffer.String(), "stdout proof") {
		t.Fatalf("stdout output missing proof text")
	}
	if !strings.Contains(stderrBuffer.String(), "stderr proof") {
		t.Fatalf("stderr output missing proof text")
	}

	httpURL := fmt.Sprintf("http://127.0.0.1:%s/health", httpHealthPort)
	tcpAddress := fmt.Sprintf("127.0.0.1:%s", tcpHealthPort)

	httpHealthy := postJSON(t, app.handleHTTPHealthAction, actionRequest{Mode: "healthy", Message: "http healthy"})
	if httpHealthy.Code != http.StatusOK {
		t.Fatalf("http-health healthy returned %d", httpHealthy.Code)
	}
	waitForDedicatedHTTPStatus(t, httpURL, http.StatusOK, `"status":"ok"`)

	httpError := postJSON(t, app.handleHTTPHealthAction, actionRequest{Mode: "error", Message: "http error"})
	if httpError.Code != http.StatusOK {
		t.Fatalf("http-health error returned %d", httpError.Code)
	}
	waitForDedicatedHTTPStatus(t, httpURL, http.StatusInternalServerError, `"status":"error"`)

	httpStopped := postJSON(t, app.handleHTTPHealthAction, actionRequest{Mode: "stopped", Message: "http stopped"})
	if httpStopped.Code != http.StatusOK {
		t.Fatalf("http-health stopped returned %d", httpStopped.Code)
	}
	waitForDedicatedHTTPUnavailable(t, httpURL)

	tcpHealthy := postJSON(t, app.handleTCPHealthAction, actionRequest{Mode: "healthy", Message: "tcp healthy"})
	if tcpHealthy.Code != http.StatusOK {
		t.Fatalf("tcp-health healthy returned %d", tcpHealthy.Code)
	}
	waitForTCPPayload(t, tcpAddress, "OK")

	tcpError := postJSON(t, app.handleTCPHealthAction, actionRequest{Mode: "error", Message: "tcp error"})
	if tcpError.Code != http.StatusOK {
		t.Fatalf("tcp-health error returned %d", tcpError.Code)
	}
	waitForTCPPayload(t, tcpAddress, "ERROR")

	tcpStopped := postJSON(t, app.handleTCPHealthAction, actionRequest{Mode: "stopped", Message: "tcp stopped"})
	if tcpStopped.Code != http.StatusOK {
		t.Fatalf("tcp-health stopped returned %d", tcpStopped.Code)
	}
	waitForTCPUnavailable(t, tcpAddress)

	logs := getJSON(t, app.handleLogs)
	if logs.Code != http.StatusOK {
		t.Fatalf("logs returned %d", logs.Code)
	}
	var logsBody map[string]any
	if err := json.Unmarshal(logs.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs body failed: %v", err)
	}
	content := logsBody["content"].(string)
	for _, needle := range []string{"stdout proof", "stderr proof", "http stopped", "tcp stopped"} {
		if !strings.Contains(content, needle) {
			t.Fatalf("expected logs to contain %q", needle)
		}
	}
}

func TestForkChildAndStartChildTrackProcesses(t *testing.T) {
	childBinary := buildHarnessBinary(t)
	app := newTestHarnessAppWithEnv(t, map[string]string{
		"ECHO_CHILD_EXECUTABLE": childBinary,
	})

	forkResponse := postJSON(t, app.handleForkChild, actionRequest{Name: "forked-child"})
	if forkResponse.Code != http.StatusOK {
		t.Fatalf("fork-child returned %d", forkResponse.Code)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		for _, child := range app.snapshot().Children {
			if child.Name == "forked-child" && child.ExitedAt != nil {
				return true
			}
		}
		return false
	})

	startResponse := postJSON(t, app.handleStartChild, actionRequest{Name: "running-child"})
	if startResponse.Code != http.StatusOK {
		t.Fatalf("start-child returned %d", startResponse.Code)
	}

	var runningPID int
	waitForCondition(t, 5*time.Second, func() bool {
		for _, child := range app.snapshot().Children {
			if child.Name == "running-child" {
				runningPID = child.PID
				return child.PID > 0
			}
		}
		return false
	})

	process, err := os.FindProcess(runningPID)
	if err != nil {
		t.Fatalf("find running child process failed: %v", err)
	}
	_ = process.Kill()

	waitForCondition(t, 5*time.Second, func() bool {
		for _, child := range app.snapshot().Children {
			if child.Name == "running-child" {
				return child.ExitedAt != nil && child.ExitCode != nil
			}
		}
		return false
	})

	for _, expected := range []string{
		filepath.Join(filepath.Dir(app.statePath), "children", "forked-child.log"),
		filepath.Join(filepath.Dir(app.statePath), "children", "forked-child.state.json"),
		filepath.Join(filepath.Dir(app.statePath), "children", "forked-child.sqlite"),
	} {
		expectedPath := expected
		waitForCondition(t, 5*time.Second, func() bool {
			_, err := os.Stat(expectedPath)
			return err == nil
		})
	}
}

func TestRunServesHTTPAndClosesGracefully(t *testing.T) {
	app := newTestHarnessAppWithEnv(t, map[string]string{
		"ECHO_HTTP_HEALTH_PORT": findFreePort(t),
		"ECHO_TCP_PORT":         findFreePort(t),
	})
	app.port = findFreePort(t)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- app.run()
	}()

	baseURL := "http://127.0.0.1:" + app.port
	waitForHTTPReady(t, baseURL)

	response, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer response.Body.Close()
	page, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read index page failed: %v", err)
	}
	if !strings.Contains(string(page), "Echo Service Harness") {
		t.Fatalf("unexpected index page contents")
	}

	closeResponse, err := http.Post(baseURL+"/action/close", "application/json", strings.NewReader(`{"message":"test-close"}`))
	if err != nil {
		t.Fatalf("POST /action/close failed: %v", err)
	}
	closeResponse.Body.Close()

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("server did not shut down after close action")
	}
}

func TestSubprocessCloseAndAbortFlows(t *testing.T) {
	binary := buildHarnessBinary(t)

	t.Run("close", func(t *testing.T) {
		logPath, statePath, dbPath := makeTestPaths(t)
		port := findFreePort(t)
		command := startHarnessProcess(t, binary, map[string]string{
			"ECHO_PORT":             port,
			"ECHO_LOG_PATH":         logPath,
			"ECHO_STATE_PATH":       statePath,
			"ECHO_DB_PATH":          dbPath,
			"ECHO_HTTP_HEALTH_PORT": findFreePort(t),
			"ECHO_TCP_PORT":         findFreePort(t),
		})

		baseURL := "http://127.0.0.1:" + port
		waitForHTTPReady(t, baseURL)

		response, err := http.Post(baseURL+"/action/close", "application/json", strings.NewReader(`{"message":"close-from-test"}`))
		if err != nil {
			t.Fatalf("POST close failed: %v", err)
		}
		response.Body.Close()

		done := make(chan error, 1)
		go func() { done <- command.Wait() }()

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("close subprocess exited with error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("close subprocess did not exit in time")
		}
	})

	t.Run("abort", func(t *testing.T) {
		logPath, statePath, dbPath := makeTestPaths(t)
		port := findFreePort(t)
		command := startHarnessProcess(t, binary, map[string]string{
			"ECHO_PORT":             port,
			"ECHO_LOG_PATH":         logPath,
			"ECHO_STATE_PATH":       statePath,
			"ECHO_DB_PATH":          dbPath,
			"ECHO_HTTP_HEALTH_PORT": findFreePort(t),
			"ECHO_TCP_PORT":         findFreePort(t),
		})

		baseURL := "http://127.0.0.1:" + port
		waitForHTTPReady(t, baseURL)

		response, err := http.Post(baseURL+"/action/abort", "application/json", strings.NewReader(`{"message":"abort-from-test"}`))
		if err != nil {
			t.Fatalf("POST abort failed: %v", err)
		}
		response.Body.Close()

		done := make(chan error, 1)
		go func() { done <- command.Wait() }()

		select {
		case err := <-done:
			if err == nil {
				t.Fatalf("abort subprocess exited cleanly; expected non-zero exit")
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("abort returned unexpected error type: %T", err)
			}
			if exitErr.ExitCode() == 0 {
				t.Fatalf("abort exit code should be non-zero")
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("abort subprocess did not exit in time")
		}
	})
}

func TestServiceManifestMatchesHarnessContract(t *testing.T) {
	rawManifest, err := os.ReadFile("service.json")
	if err != nil {
		t.Fatalf("read service.json failed: %v", err)
	}

	var manifest struct {
		ID           string            `json:"id"`
		Name         string            `json:"name"`
		Executable   string            `json:"executable"`
		Args         []string          `json:"args"`
		Env          map[string]string `json:"env"`
		Healthchecks []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"healthchecks"`
	}
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("decode service.json failed: %v", err)
	}

	if manifest.ID != "echo-service" {
		t.Fatalf("unexpected manifest id: %s", manifest.ID)
	}
	if manifest.Executable != "go" {
		t.Fatalf("unexpected executable: %s", manifest.Executable)
	}
	if len(manifest.Args) != 2 || manifest.Args[0] != "run" || manifest.Args[1] != "." {
		t.Fatalf("unexpected args: %#v", manifest.Args)
	}
	if len(manifest.Healthchecks) != 1 {
		t.Fatalf("expected one healthchecks[] item, got %d", len(manifest.Healthchecks))
	}
	if manifest.Healthchecks[0].ID != "process-ready" {
		t.Fatalf("unexpected healthchecks[] id: %s", manifest.Healthchecks[0].ID)
	}
	if manifest.Healthchecks[0].Type != "process" {
		t.Fatalf("unexpected healthchecks[] type: %s", manifest.Healthchecks[0].Type)
	}
	requiredEnv := []string{"ECHO_PORT", "ECHO_LOG_PATH", "ECHO_STATE_PATH", "ECHO_DB_PATH"}
	requiredEnv = append(requiredEnv, "ECHO_HTTP_HEALTH_PORT", "ECHO_TCP_PORT")
	for _, key := range requiredEnv {
		if manifest.Env[key] == "" {
			t.Fatalf("missing required env %s", key)
		}
	}

	rawContract, err := os.ReadFile(filepath.Join("verify", "service-harness.json"))
	if err != nil {
		t.Fatalf("read service-harness.json failed: %v", err)
	}
	var contract struct {
		ServiceID string `json:"serviceId"`
		Health    struct {
			Type string `json:"type"`
		} `json:"health"`
		API struct {
			Endpoints []string `json:"endpoints"`
			Actions   []string `json:"actions"`
		} `json:"api"`
	}
	if err := json.Unmarshal(rawContract, &contract); err != nil {
		t.Fatalf("decode service-harness.json failed: %v", err)
	}
	if contract.ServiceID != "echo-service" {
		t.Fatalf("unexpected contract serviceId: %s", contract.ServiceID)
	}
	if contract.Health.Type != "process" {
		t.Fatalf("unexpected contract health type: %s", contract.Health.Type)
	}
	for _, endpoint := range []string{"/env", "/global-env", "/service-lasso/output", "/health/http", "/health/tcp"} {
		if !containsString(contract.API.Endpoints, endpoint) {
			t.Fatalf("missing contract endpoint %s", endpoint)
		}
	}
	for _, action := range []string{"write-stdout", "write-stderr", "http-health", "tcp-health"} {
		if !containsString(contract.API.Actions, action) {
			t.Fatalf("missing contract action %s", action)
		}
	}
}

func TestSQLiteDatabaseCanBeOpenedAfterWrites(t *testing.T) {
	app := newTestHarnessApp(t)
	postJSON(t, app.handleWriteSQLite, actionRequest{Message: "db-proof"})
	_ = app.db.Close()
	app.db = nil

	db, err := sql.Open("sqlite", app.dbPath)
	if err != nil {
		t.Fatalf("re-open sqlite database failed: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM events WHERE detail = ?`, "db-proof").Scan(&count); err != nil {
		t.Fatalf("query sqlite proof row failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one sqlite proof row, got %d", count)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
