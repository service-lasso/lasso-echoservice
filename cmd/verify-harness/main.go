package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type contract struct {
	ServiceID string `json:"serviceId"`
	Health    struct {
		Type           string `json:"type"`
		TimeoutSeconds int    `json:"timeoutSeconds"`
	} `json:"health"`
	Expect struct {
		Logs      bool `json:"logs"`
		State     bool `json:"state"`
		SQLite    bool `json:"sqlite"`
		Stdout    bool `json:"stdout"`
		Stderr    bool `json:"stderr"`
		ExitClean bool `json:"exitClean"`
	} `json:"expect"`
	API struct {
		Endpoints []string `json:"endpoints"`
		Actions   []string `json:"actions"`
	} `json:"api"`
	Artifacts struct {
		CaptureLogs    bool `json:"captureLogs"`
		CaptureState   bool `json:"captureState"`
		CaptureSummary bool `json:"captureSummary"`
	} `json:"artifacts"`
}

type summary struct {
	BinaryPath   string            `json:"binaryPath"`
	BaseURL      string            `json:"baseUrl"`
	ServiceID    string            `json:"serviceId"`
	Verified     []string          `json:"verified"`
	Files        map[string]string `json:"files"`
	SQLiteRows   int               `json:"sqliteRows"`
	LastAction   string            `json:"lastAction"`
	ChildActions []string          `json:"childActions"`
	HealthTargets map[string]string `json:"healthTargets"`
}

type stateResponse struct {
	Service      string `json:"service"`
	LastAction   string `json:"lastAction"`
	ActionCount  int    `json:"actionCount"`
	LogPath      string `json:"logPath"`
	StatePath    string `json:"statePath"`
	DatabasePath string `json:"databasePath"`
	Children     []struct {
		Name     string     `json:"name"`
		PID      int        `json:"pid"`
		ExitedAt *time.Time `json:"exitedAt"`
		ExitCode *int       `json:"exitCode"`
	} `json:"children"`
}

type verifierTargets struct {
	httpHealthURL string
	tcpAddress    string
}

func main() {
	contractPath := flag.String("contract", "./verify/service-harness.json", "path to verification contract")
	outputDir := flag.String("output-dir", "./output/verify", "output directory for verification artifacts")
	flag.Parse()

	if err := run(*contractPath, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "verify-harness failed: %v\n", err)
		os.Exit(1)
	}
}

func run(contractPath, outputDir string) error {
	doc, err := loadContract(contractPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	binaryPath, err := buildBinary(outputDir)
	if err != nil {
		return err
	}

	port, err := findFreePort()
	if err != nil {
		return err
	}
	httpHealthPort, err := findFreePort()
	if err != nil {
		return err
	}
	tcpPort, err := findFreePort()
	if err != nil {
		return err
	}

	runtimeRoot := filepath.Join(outputDir, "runtime")
	if err := os.MkdirAll(runtimeRoot, 0o755); err != nil {
		return fmt.Errorf("create runtime output dir: %w", err)
	}

	logPath := filepath.Join(runtimeRoot, "echo.log")
	statePath := filepath.Join(runtimeRoot, "state.json")
	dbPath := filepath.Join(runtimeRoot, "echo.sqlite")
	stdoutPath := filepath.Join(outputDir, "stdout.log")
	stderrPath := filepath.Join(outputDir, "stderr.log")

	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return fmt.Errorf("create stdout log: %w", err)
	}
	defer stdoutFile.Close()

	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		return fmt.Errorf("create stderr log: %w", err)
	}
	defer stderrFile.Close()

	cmd := exec.Command(binaryPath)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	cmd.Env = append(os.Environ(),
		"ECHO_MESSAGE=verify-harness",
		"ECHO_PORT="+port,
		"ECHO_LOG_PATH="+logPath,
		"ECHO_STATE_PATH="+statePath,
		"ECHO_DB_PATH="+dbPath,
		"ECHO_HTTP_HEALTH_PORT="+httpHealthPort,
		"ECHO_TCP_PORT="+tcpPort,
		`SERVICE_LASSO_GLOBAL_ENV_JSON={"VERIFY":"true","CHANNEL":"ci"}`,
		"SERVICE_LASSO_GLOBAL_REGION=ci",
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start harness binary: %w", err)
	}

	baseURL := "http://127.0.0.1:" + port
	targets := verifierTargets{
		httpHealthURL: fmt.Sprintf("http://127.0.0.1:%s/health", httpHealthPort),
		tcpAddress:    fmt.Sprintf("127.0.0.1:%s", tcpPort),
	}

	timeout := time.Duration(doc.Health.TimeoutSeconds)
	if timeout <= 0 {
		timeout = 30
	}
	if err := waitForHealthy(baseURL, timeout*time.Second); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}

	verified := make([]string, 0, len(doc.API.Endpoints)+len(doc.API.Actions)+8)
	for _, endpoint := range doc.API.Endpoints {
		if err := verifyEndpoint(baseURL, endpoint); err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return err
		}
		verified = append(verified, "endpoint:"+endpoint)
	}

	childActions := []string{}
	for _, action := range doc.API.Actions {
		childInfo, err := verifyAction(baseURL, targets, action)
		if err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return err
		}
		if childInfo != "" {
			childActions = append(childActions, childInfo)
		}
		verified = append(verified, "action:"+action)
	}

	state, err := fetchState(baseURL)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}

	if state.Service != doc.ServiceID {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return fmt.Errorf("state serviceId mismatch: expected %s got %s", doc.ServiceID, state.Service)
	}

	sqliteRows := 0
	if doc.Expect.SQLite {
		sqliteRows, err = countSQLiteRows(dbPath)
		if err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return err
		}
		if sqliteRows == 0 {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return fmt.Errorf("expected sqlite rows but found none")
		}
		verified = append(verified, "sqlite")
	}

	if doc.Expect.Logs {
		if err := ensureFileContains(logPath, []string{
			"verify-log",
			"verify-state",
			"verify-sqlite",
			"verify-stdout",
			"verify-stderr",
			"http-health",
			"tcp-health",
		}); err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return err
		}
		verified = append(verified, "logs")
	}

	if doc.Expect.State {
		if _, err := os.Stat(statePath); err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return fmt.Errorf("state file missing: %w", err)
		}
		verified = append(verified, "state")
	}

	if doc.Expect.Stdout {
		if err := ensureFileContains(stdoutPath, []string{"verify-stdout"}); err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return err
		}
		verified = append(verified, "stdout")
	}

	if doc.Expect.Stderr {
		if err := ensureFileContains(stderrPath, []string{"verify-stderr"}); err != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return err
		}
		verified = append(verified, "stderr")
	}

	if err := closeHarness(baseURL); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}

	waitErr := waitWithTimeout(cmd, 10*time.Second)
	if doc.Expect.ExitClean && waitErr != nil {
		return fmt.Errorf("expected clean exit after close: %w", waitErr)
	}
	verified = append(verified, "close")

	if doc.Artifacts.CaptureSummary {
		report := summary{
			BinaryPath:   binaryPath,
			BaseURL:      baseURL,
			ServiceID:    doc.ServiceID,
			Verified:     verified,
			Files:        map[string]string{"log": logPath, "state": statePath, "sqlite": dbPath, "stdout": stdoutPath, "stderr": stderrPath},
			SQLiteRows:   sqliteRows,
			LastAction:   state.LastAction,
			ChildActions: childActions,
			HealthTargets: map[string]string{"http": targets.httpHealthURL, "tcp": targets.tcpAddress},
		}
		if err := writeJSON(filepath.Join(outputDir, "summary.json"), report); err != nil {
			return err
		}
	}

	return nil
}

func loadContract(path string) (contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return contract{}, fmt.Errorf("read contract: %w", err)
	}
	var doc contract
	if err := json.Unmarshal(raw, &doc); err != nil {
		return contract{}, fmt.Errorf("decode contract: %w", err)
	}
	return doc, nil
}

func buildBinary(outputDir string) (string, error) {
	binaryName := "echo-service-verify"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(outputDir, binaryName)

	command := exec.Command("go", "build", "-o", binaryPath, ".")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w\n%s", err, stderr.String())
	}

	return binaryPath, nil
}

func findFreePort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("find free port: %w", err)
	}
	defer listener.Close()
	return fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port), nil
}

func waitForHealthy(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := http.Get(baseURL + "/health")
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr == nil && response.StatusCode == http.StatusOK && strings.Contains(string(body), `"status":"ok"`) {
				return nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("health endpoint did not become ready at %s within %s", baseURL, timeout)
}

func verifyEndpoint(baseURL, endpoint string) error {
	response, err := http.Get(baseURL + endpoint)
	if err != nil {
		return fmt.Errorf("GET %s failed: %w", endpoint, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", endpoint, response.StatusCode)
	}
	return nil
}

func verifyAction(baseURL string, targets verifierTargets, action string) (string, error) {
	switch action {
	case "write-log":
		return "", postActionExpect(baseURL, action, `{"message":"verify-log"}`, http.StatusOK)
	case "write-state":
		return "", postActionExpect(baseURL, action, `{"message":"verify-state"}`, http.StatusOK)
	case "write-sqlite":
		return "", postActionExpect(baseURL, action, `{"message":"verify-sqlite"}`, http.StatusOK)
	case "write-stdout":
		return "", postActionExpect(baseURL, action, `{"message":"verify-stdout"}`, http.StatusOK)
	case "write-stderr":
		return "", postActionExpect(baseURL, action, `{"message":"verify-stderr"}`, http.StatusOK)
	case "error":
		return "", postActionExpect(baseURL, action, `{"message":"verify-error"}`, http.StatusInternalServerError)
	case "fork-child":
		if err := postActionExpect(baseURL, action, `{"name":"verify-fork-child"}`, http.StatusOK); err != nil {
			return "", err
		}
		state, err := fetchState(baseURL)
		if err != nil {
			return "", err
		}
		for _, child := range state.Children {
			if child.Name == "verify-fork-child" {
				return "fork-child", waitForChildExit(baseURL, child.Name, 5*time.Second)
			}
		}
		return "", fmt.Errorf("fork-child did not appear in state children")
	case "start-child":
		if err := postActionExpect(baseURL, action, `{"name":"verify-running-child"}`, http.StatusOK); err != nil {
			return "", err
		}
		state, err := fetchState(baseURL)
		if err != nil {
			return "", err
		}
		for _, child := range state.Children {
			if child.Name == "verify-running-child" {
				if child.PID <= 0 {
					return "", fmt.Errorf("start-child did not return a valid pid")
				}
				process, err := os.FindProcess(child.PID)
				if err != nil {
					return "", fmt.Errorf("find running child process: %w", err)
				}
				_ = process.Kill()
				return "start-child", waitForChildExit(baseURL, child.Name, 5*time.Second)
			}
		}
		return "", fmt.Errorf("start-child did not appear in state children")
	case "http-health":
		if err := postActionExpect(baseURL, action, `{"mode":"healthy","message":"verify http healthy"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForDedicatedHTTPStatus(targets.httpHealthURL, http.StatusOK, `"status":"ok"`); err != nil {
			return "", err
		}
		if err := postActionExpect(baseURL, action, `{"mode":"error","message":"verify http error"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForDedicatedHTTPStatus(targets.httpHealthURL, http.StatusInternalServerError, `"status":"error"`); err != nil {
			return "", err
		}
		if err := postActionExpect(baseURL, action, `{"mode":"stopped","message":"verify http stopped"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForDedicatedHTTPUnavailable(targets.httpHealthURL); err != nil {
			return "", err
		}
		return "http-health", postActionExpect(baseURL, action, `{"mode":"healthy","message":"verify http restore"}`, http.StatusOK)
	case "tcp-health":
		if err := postActionExpect(baseURL, action, `{"mode":"healthy","message":"verify tcp healthy"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForTCPPayload(targets.tcpAddress, "OK"); err != nil {
			return "", err
		}
		if err := postActionExpect(baseURL, action, `{"mode":"error","message":"verify tcp error"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForTCPPayload(targets.tcpAddress, "ERROR"); err != nil {
			return "", err
		}
		if err := postActionExpect(baseURL, action, `{"mode":"stopped","message":"verify tcp stopped"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForTCPUnavailable(targets.tcpAddress); err != nil {
			return "", err
		}
		if err := postActionExpect(baseURL, action, `{"mode":"healthy","message":"verify tcp restore"}`, http.StatusOK); err != nil {
			return "", err
		}
		if err := waitForTCPPayload(targets.tcpAddress, "OK"); err != nil {
			return "", err
		}
		return "tcp-health", nil
	default:
		return "", fmt.Errorf("unsupported verification action: %s", action)
	}
}

func postActionExpect(baseURL, action, body string, expectedStatus int) error {
	response, err := http.Post(baseURL+"/action/"+action, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST /action/%s failed: %w", action, err)
	}
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		return fmt.Errorf("POST /action/%s returned %d, expected %d", action, response.StatusCode, expectedStatus)
	}
	return nil
}

func fetchState(baseURL string) (stateResponse, error) {
	response, err := http.Get(baseURL + "/state")
	if err != nil {
		return stateResponse{}, fmt.Errorf("GET /state failed: %w", err)
	}
	defer response.Body.Close()

	var body stateResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return stateResponse{}, fmt.Errorf("decode /state failed: %w", err)
	}
	return body, nil
}

func waitForChildExit(baseURL, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := fetchState(baseURL)
		if err == nil {
			for _, child := range state.Children {
				if child.Name == name && child.ExitedAt != nil {
					return nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("child %s did not exit within %s", name, timeout)
}

func waitForDedicatedHTTPStatus(url string, expectedStatus int, expectedText string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr == nil && response.StatusCode == expectedStatus && strings.Contains(string(body), expectedText) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("dedicated HTTP health %s did not return %d with %q", url, expectedStatus, expectedText)
}

func waitForDedicatedHTTPUnavailable(url string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, err := http.Get(url)
		if err != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("dedicated HTTP health %s remained reachable", url)
}

func waitForTCPPayload(address string, expected string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		_ = connection.SetDeadline(time.Now().Add(time.Second))
		body, readErr := io.ReadAll(connection)
		connection.Close()
		if readErr == nil && strings.Contains(string(body), expected) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("TCP health %s did not yield %q", address, expected)
}

func waitForTCPUnavailable(address string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 250*time.Millisecond)
		if err != nil {
			return nil
		}
		connection.Close()
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("TCP health %s remained reachable", address)
}

func countSQLiteRows(path string) (int, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return 0, fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count sqlite rows: %w", err)
	}
	return count, nil
}

func ensureFileContains(path string, needles []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}
	text := string(raw)
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return fmt.Errorf("file %s missing expected content %q", path, needle)
		}
	}
	return nil
}

func closeHarness(baseURL string) error {
	response, err := http.Post(baseURL+"/action/close", "application/json", strings.NewReader(`{"message":"verify-close"}`))
	if err != nil {
		return fmt.Errorf("POST /action/close failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("POST /action/close returned %d", response.StatusCode)
	}
	return nil
}

func waitWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("process did not exit within %s", timeout)
	}
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}
