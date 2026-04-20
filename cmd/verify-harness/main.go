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

type sqliteResponse struct {
	Events []struct {
		Kind   string `json:"kind"`
		Detail string `json:"detail"`
	} `json:"events"`
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
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start harness binary: %w", err)
	}

	baseURL := "http://127.0.0.1:" + port
	timeout := time.Duration(doc.Health.TimeoutSeconds)
	if timeout <= 0 {
		timeout = 30
	}
	if err := waitForHealthy(baseURL, timeout*time.Second); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}

	verified := make([]string, 0, len(doc.API.Endpoints)+len(doc.API.Actions)+6)
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
		childInfo, err := verifyAction(baseURL, action)
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
		if err := ensureFileContains(logPath, []string{"verify-log", "verify-state", "verify-sqlite"}); err != nil {
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

func verifyAction(baseURL, action string) (string, error) {
	var body string
	switch action {
	case "write-log":
		body = `{"message":"verify-log"}`
	case "write-state":
		body = `{"message":"verify-state"}`
	case "write-sqlite":
		body = `{"message":"verify-sqlite"}`
	case "fork-child":
		body = `{"name":"verify-fork-child"}`
	case "start-child":
		body = `{"name":"verify-running-child"}`
	case "error":
		body = `{"message":"verify-error"}`
	default:
		return "", fmt.Errorf("unsupported verification action: %s", action)
	}

	response, err := http.Post(baseURL+"/action/"+action, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("POST /action/%s failed: %w", action, err)
	}
	defer response.Body.Close()

	if action == "error" {
		if response.StatusCode != http.StatusInternalServerError {
			return "", fmt.Errorf("POST /action/%s returned %d, expected 500", action, response.StatusCode)
		}
		return "", nil
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST /action/%s returned %d", action, response.StatusCode)
	}

	if action == "fork-child" || action == "start-child" {
		state, err := fetchState(baseURL)
		if err != nil {
			return "", err
		}
		for _, child := range state.Children {
			if child.Name == "verify-fork-child" && action == "fork-child" {
				return "fork-child", waitForChildExit(baseURL, child.Name, 5*time.Second)
			}
			if child.Name == "verify-running-child" && action == "start-child" {
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
		return "", fmt.Errorf("action %s did not appear in state children", action)
	}

	return "", nil
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
