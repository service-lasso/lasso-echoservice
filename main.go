package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	_ "modernc.org/sqlite"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type healthMode string

const (
	healthHealthy healthMode = "healthy"
	healthStopped healthMode = "stopped"
	healthError   healthMode = "error"
)

type childProcess struct {
	Name      string     `json:"name"`
	Mode      string     `json:"mode"`
	PID       int        `json:"pid"`
	StartedAt time.Time  `json:"startedAt"`
	ExitedAt  *time.Time `json:"exitedAt,omitempty"`
	ExitCode  *int       `json:"exitCode,omitempty"`
}

type eventRow struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

type harnessSnapshot struct {
	Service         string            `json:"service"`
	PID             int               `json:"pid"`
	Message         string            `json:"message"`
	StartedAt       time.Time         `json:"startedAt"`
	LastAction      string            `json:"lastAction"`
	ActionCount     int               `json:"actionCount"`
	LastError       string            `json:"lastError,omitempty"`
	LogPath         string            `json:"logPath"`
	StatePath       string            `json:"statePath"`
	DatabasePath    string            `json:"databasePath"`
	HTTPHealthPort  string            `json:"httpHealthPort"`
	HTTPHealthURL   string            `json:"httpHealthUrl"`
	HTTPHealthMode  string            `json:"httpHealthMode"`
	TCPHealthPort   string            `json:"tcpHealthPort"`
	TCPHealthAddr   string            `json:"tcpHealthAddress"`
	TCPHealthMode   string            `json:"tcpHealthMode"`
	GlobalEnv       map[string]string `json:"globalEnv"`
	Children        []childProcess    `json:"children"`
	RecentEvents    []eventRow        `json:"recentEvents"`
	ShutdownMode    string            `json:"shutdownMode,omitempty"`
}

type actionRequest struct {
	Message string `json:"message"`
	Name    string `json:"name"`
	Mode    string `json:"mode"`
}

type harnessApp struct {
	serviceName string
	message     string
	port        string
	logPath     string
	statePath   string
	dbPath      string

	httpHealthPort string
	tcpHealthPort  string

	startedAt time.Time
	globalEnv map[string]string

	mu             sync.Mutex
	lastAction     string
	actionCount    int
	lastError      string
	children       []childProcess
	recentEvents   []eventRow
	httpHealthMode healthMode
	tcpHealthMode  healthMode

	logFile         *os.File
	db              *sql.DB
	httpServer      *http.Server
	httpHealthServer *http.Server
	tcpHealthLn     net.Listener
	stopCh          chan string
	stdout          io.Writer
	stderr          io.Writer
}

func main() {
	childMode := flag.Bool("child", false, "run as a child harness process")
	oneShot := flag.Bool("one-shot", false, "write one event and exit")
	childName := flag.String("child-name", "child", "child process name")
	flag.Parse()

	if *childMode {
		if err := runChild(*childName, *oneShot); err != nil {
			log.Printf("echo-service child failed: %v", err)
			os.Exit(2)
		}
		return
	}

	app, err := newHarnessApp()
	if err != nil {
		log.Printf("echo-service failed to initialize: %v", err)
		os.Exit(1)
	}

	if err := app.run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("echo-service failed: %v", err)
		os.Exit(1)
	}
}

func newHarnessApp() (*harnessApp, error) {
	logPath := envOrDefault("ECHO_LOG_PATH", "./runtime/echo.log")
	statePath := envOrDefault("ECHO_STATE_PATH", "./runtime/state.json")
	dbPath := envOrDefault("ECHO_DB_PATH", "./runtime/echo.sqlite")

	for _, candidate := range []string{logPath, statePath, dbPath} {
		if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
			return nil, fmt.Errorf("create runtime directory: %w", err)
		}
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("create sqlite schema: %w", err)
	}

	app := &harnessApp{
		serviceName:    "echo-service",
		message:        envOrDefault("ECHO_MESSAGE", "hello from echo-service harness"),
		port:           envOrDefault("ECHO_PORT", "4010"),
		logPath:        logPath,
		statePath:      statePath,
		dbPath:         dbPath,
		httpHealthPort: envOrDefault("ECHO_HTTP_HEALTH_PORT", "4011"),
		tcpHealthPort:  envOrDefault("ECHO_TCP_PORT", "4012"),
		startedAt:      time.Now().UTC(),
		globalEnv:      collectGlobalEnv(),
		logFile:        logFile,
		db:             db,
		stopCh:         make(chan string, 1),
		stdout:         os.Stdout,
		stderr:         os.Stderr,
		httpHealthMode: parseHealthMode(os.Getenv("ECHO_HTTP_HEALTH_MODE")),
		tcpHealthMode:  parseHealthMode(os.Getenv("ECHO_TCP_HEALTH_MODE")),
	}

	if err := app.recordEvent("startup", app.message); err != nil {
		return nil, err
	}

	return app, nil
}

func (app *harnessApp) run() error {
	if err := app.setHTTPHealthMode(app.currentHTTPHealthMode()); err != nil {
		return err
	}
	if err := app.setTCPHealthMode(app.currentTCPHealthMode()); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleIndex)
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/health/http", app.handleHTTPHealthStatus)
	mux.HandleFunc("/health/tcp", app.handleTCPHealthStatus)
	mux.HandleFunc("/state", app.handleState)
	mux.HandleFunc("/logs", app.handleLogs)
	mux.HandleFunc("/sqlite", app.handleSQLite)
	mux.HandleFunc("/env", app.handleEnv)
	mux.HandleFunc("/global-env", app.handleGlobalEnv)
	mux.HandleFunc("/service-lasso/output", app.handleServiceLassoOutput)
	mux.HandleFunc("/action/write-log", app.handleWriteLog)
	mux.HandleFunc("/action/write-state", app.handleWriteState)
	mux.HandleFunc("/action/write-sqlite", app.handleWriteSQLite)
	mux.HandleFunc("/action/write-stdout", app.handleWriteStdout)
	mux.HandleFunc("/action/write-stderr", app.handleWriteStderr)
	mux.HandleFunc("/action/http-health", app.handleHTTPHealthAction)
	mux.HandleFunc("/action/tcp-health", app.handleTCPHealthAction)
	mux.HandleFunc("/action/error", app.handleErrorAction)
	mux.HandleFunc("/action/close", app.handleClose)
	mux.HandleFunc("/action/abort", app.handleAbort)
	mux.HandleFunc("/action/start-child", app.handleStartChild)
	mux.HandleFunc("/action/fork-child", app.handleForkChild)

	app.httpServer = &http.Server{
		Addr:              "127.0.0.1:" + app.port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- app.httpServer.ListenAndServe()
	}()

	ctx, stop := signalNotifyContext()
	defer stop()

	select {
	case reason := <-app.stopCh:
		_ = app.shutdown(reason)
		return nil
	case <-ctx.Done():
		_ = app.shutdown("signal")
		return nil
	case err := <-serverErrCh:
		return err
	}
}

func (app *harnessApp) shutdown(reason string) error {
	app.mu.Lock()
	app.lastAction = reason
	app.mu.Unlock()

	_ = app.recordEvent("shutdown", reason)
	app.closeHealthServers()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if app.httpServer != nil {
		_ = app.httpServer.Shutdown(shutdownCtx)
	}

	if app.logFile != nil {
		if err := app.logFile.Close(); err != nil {
			return err
		}
		app.logFile = nil
	}

	if app.db != nil {
		if err := app.db.Close(); err != nil {
			return err
		}
		app.db = nil
	}

	return nil
}

func (app *harnessApp) snapshot() harnessSnapshot {
	app.mu.Lock()
	defer app.mu.Unlock()

	children := append([]childProcess(nil), app.children...)
	recent := append([]eventRow(nil), app.recentEvents...)
	globalEnv := cloneStringMap(app.globalEnv)

	return harnessSnapshot{
		Service:        app.serviceName,
		PID:            os.Getpid(),
		Message:        app.message,
		StartedAt:      app.startedAt,
		LastAction:     app.lastAction,
		ActionCount:    app.actionCount,
		LastError:      app.lastError,
		LogPath:        app.logPath,
		StatePath:      app.statePath,
		DatabasePath:   app.dbPath,
		HTTPHealthPort: app.httpHealthPort,
		HTTPHealthURL:  fmt.Sprintf("http://127.0.0.1:%s/health", app.httpHealthPort),
		HTTPHealthMode: string(app.httpHealthMode),
		TCPHealthPort:  app.tcpHealthPort,
		TCPHealthAddr:  fmt.Sprintf("127.0.0.1:%s", app.tcpHealthPort),
		TCPHealthMode:  string(app.tcpHealthMode),
		GlobalEnv:      globalEnv,
		Children:       children,
		RecentEvents:   recent,
	}
}

func (app *harnessApp) persistState() error {
	snapshot := app.snapshot()
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(app.statePath, payload, 0o644)
}

func (app *harnessApp) recordEvent(kind, detail string) error {
	app.mu.Lock()
	app.lastAction = kind
	app.actionCount++
	app.mu.Unlock()

	now := time.Now().UTC()
	if _, err := fmt.Fprintf(app.logFile, "%s [%s] %s\n", now.Format(time.RFC3339), kind, detail); err != nil {
		return err
	}

	result, err := app.db.Exec(`INSERT INTO events (kind, detail, created_at) VALUES (?, ?, ?)`, kind, detail, now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()

	app.mu.Lock()
	app.recentEvents = append([]eventRow{{
		ID:        id,
		Kind:      kind,
		Detail:    detail,
		CreatedAt: now,
	}}, app.recentEvents...)
	if len(app.recentEvents) > 20 {
		app.recentEvents = app.recentEvents[:20]
	}
	app.mu.Unlock()

	return app.persistState()
}

func (app *harnessApp) setLastError(message string) {
	app.mu.Lock()
	app.lastError = message
	app.mu.Unlock()
}

func (app *harnessApp) readActionRequest(r *http.Request) actionRequest {
	if r.Body == nil {
		return actionRequest{}
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || len(body) == 0 {
		return actionRequest{}
	}

	var req actionRequest
	if json.Unmarshal(body, &req) == nil {
		return req
	}

	return actionRequest{}
}

func (app *harnessApp) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (app *harnessApp) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, indexHTML)
}

func (app *harnessApp) handleHealth(w http.ResponseWriter, _ *http.Request) {
	snapshot := app.snapshot()
	app.writeJSON(w, http.StatusOK, map[string]any{
		"service":        app.serviceName,
		"status":         "ok",
		"pid":            os.Getpid(),
		"uptime":         time.Since(app.startedAt).String(),
		"httpHealthMode": snapshot.HTTPHealthMode,
		"httpHealthUrl":  snapshot.HTTPHealthURL,
		"tcpHealthMode":  snapshot.TCPHealthMode,
		"tcpHealthAddr":  snapshot.TCPHealthAddr,
	})
}

func (app *harnessApp) handleHTTPHealthStatus(w http.ResponseWriter, _ *http.Request) {
	snapshot := app.snapshot()
	app.writeJSON(w, http.StatusOK, map[string]any{
		"service": app.serviceName,
		"mode":    snapshot.HTTPHealthMode,
		"url":     snapshot.HTTPHealthURL,
		"port":    snapshot.HTTPHealthPort,
	})
}

func (app *harnessApp) handleTCPHealthStatus(w http.ResponseWriter, _ *http.Request) {
	snapshot := app.snapshot()
	app.writeJSON(w, http.StatusOK, map[string]any{
		"service": app.serviceName,
		"mode":    snapshot.TCPHealthMode,
		"address": snapshot.TCPHealthAddr,
		"port":    snapshot.TCPHealthPort,
	})
}

func (app *harnessApp) handleState(w http.ResponseWriter, _ *http.Request) {
	app.writeJSON(w, http.StatusOK, app.snapshot())
}

func (app *harnessApp) handleLogs(w http.ResponseWriter, _ *http.Request) {
	content, err := os.ReadFile(app.logPath)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{
		"path":    app.logPath,
		"content": string(content),
	})
}

func (app *harnessApp) handleSQLite(w http.ResponseWriter, _ *http.Request) {
	rows, err := app.db.Query(`SELECT id, kind, detail, created_at FROM events ORDER BY id DESC LIMIT 20`)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()

	var events []eventRow
	for rows.Next() {
		var item eventRow
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Kind, &item.Detail, &createdAt); err != nil {
			app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		events = append(events, item)
	}

	app.writeJSON(w, http.StatusOK, map[string]any{
		"path":   app.dbPath,
		"events": events,
	})
}

func (app *harnessApp) handleEnv(w http.ResponseWriter, _ *http.Request) {
	app.writeJSON(w, http.StatusOK, map[string]any{
		"service":      app.serviceName,
		"env":          collectEnv(),
		"serviceEnv":   app.serviceEnv(),
		"globalEnv":    cloneStringMap(app.globalEnv),
		"serviceLasso": app.serviceLassoOutput(),
	})
}

func (app *harnessApp) handleGlobalEnv(w http.ResponseWriter, _ *http.Request) {
	output := app.serviceLassoOutput()
	app.writeJSON(w, http.StatusOK, map[string]any{
		"service":    app.serviceName,
		"globalEnv":  cloneStringMap(app.globalEnv),
		"globalenv":  cloneStringMap(app.globalEnv),
		"serviceLassoOutput": output,
	})
}

func (app *harnessApp) handleServiceLassoOutput(w http.ResponseWriter, _ *http.Request) {
	app.writeJSON(w, http.StatusOK, app.serviceLassoOutput())
}

func (app *harnessApp) handleWriteLog(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "write-log")

	if err := app.recordEvent("write-log", message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "write-log", "message": message})
}

func (app *harnessApp) handleWriteState(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "write-state")

	if err := app.recordEvent("write-state", message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "write-state", "path": app.statePath})
}

func (app *harnessApp) handleWriteSQLite(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "write-sqlite")

	if err := app.recordEvent("write-sqlite", message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "write-sqlite", "path": app.dbPath})
}

func (app *harnessApp) handleWriteStdout(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "write-stdout")

	if _, err := fmt.Fprintf(app.stdout, "%s [stdout] %s\n", time.Now().UTC().Format(time.RFC3339), message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := app.recordEvent("write-stdout", message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "write-stdout", "message": message})
}

func (app *harnessApp) handleWriteStderr(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "write-stderr")

	if _, err := fmt.Fprintf(app.stderr, "%s [stderr] %s\n", time.Now().UTC().Format(time.RFC3339), message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := app.recordEvent("write-stderr", message); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "write-stderr", "message": message})
}

func (app *harnessApp) handleHTTPHealthAction(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	mode := parseHealthMode(req.Mode)
	if err := app.setHTTPHealthMode(mode); err != nil {
		app.setLastError(err.Error())
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	message := defaultMessage(req.Message, "http-health:"+string(mode))
	if err := app.recordEvent("http-health", fmt.Sprintf("%s (%s)", string(mode), message)); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	snapshot := app.snapshot()
	app.writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"action": "http-health",
		"mode":   snapshot.HTTPHealthMode,
		"url":    snapshot.HTTPHealthURL,
	})
}

func (app *harnessApp) handleTCPHealthAction(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	mode := parseHealthMode(req.Mode)
	if err := app.setTCPHealthMode(mode); err != nil {
		app.setLastError(err.Error())
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	message := defaultMessage(req.Message, "tcp-health:"+string(mode))
	if err := app.recordEvent("tcp-health", fmt.Sprintf("%s (%s)", string(mode), message)); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	snapshot := app.snapshot()
	app.writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"action":  "tcp-health",
		"mode":    snapshot.TCPHealthMode,
		"address": snapshot.TCPHealthAddr,
	})
}

func (app *harnessApp) handleErrorAction(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "simulated harness error")

	app.setLastError(message)
	_ = app.recordEvent("error", message)
	app.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "action": "error", "message": message})
}

func (app *harnessApp) handleClose(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "close")

	_ = app.recordEvent("close", message)
	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "close", "message": message})
	go func() {
		time.Sleep(150 * time.Millisecond)
		app.stopCh <- "close"
	}()
}

func (app *harnessApp) handleAbort(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	message := defaultMessage(req.Message, "abort")

	_ = app.recordEvent("abort", message)
	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "abort", "message": message})
	go func() {
		time.Sleep(150 * time.Millisecond)
		os.Exit(2)
	}()
}

func (app *harnessApp) handleStartChild(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("child-%d", time.Now().UnixNano())
	}

	child, err := app.spawnChild(name, false)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "start-child", "child": child})
}

func (app *harnessApp) handleForkChild(w http.ResponseWriter, r *http.Request) {
	req := app.readActionRequest(r)
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("fork-%d", time.Now().UnixNano())
	}

	child, err := app.spawnChild(name, true)
	if err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	app.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": "fork-child", "child": child})
}

func (app *harnessApp) spawnChild(name string, oneShot bool) (childProcess, error) {
	executable, err := resolveChildExecutable()
	if err != nil {
		return childProcess{}, err
	}

	childrenDir := filepath.Join(filepath.Dir(app.statePath), "children")
	if err := os.MkdirAll(childrenDir, 0o755); err != nil {
		return childProcess{}, err
	}

	args := []string{"-child", "-child-name", name}
	mode := "long-running"
	if oneShot {
		args = append(args, "-one-shot")
		mode = "one-shot"
	}

	cmd := exec.Command(executable, args...)
	cmd.Env = append(os.Environ(),
		"ECHO_LOG_PATH="+filepath.Join(childrenDir, name+".log"),
		"ECHO_STATE_PATH="+filepath.Join(childrenDir, name+".state.json"),
		"ECHO_DB_PATH="+filepath.Join(childrenDir, name+".sqlite"),
		"ECHO_MESSAGE=child:"+name,
	)

	if err := cmd.Start(); err != nil {
		return childProcess{}, err
	}

	child := childProcess{
		Name:      name,
		Mode:      mode,
		PID:       cmd.Process.Pid,
		StartedAt: time.Now().UTC(),
	}

	app.mu.Lock()
	app.children = append(app.children, child)
	app.mu.Unlock()
	_ = app.recordEvent("child-start", fmt.Sprintf("%s pid=%d mode=%s", name, child.PID, mode))

	go func() {
		err := cmd.Wait()
		app.mu.Lock()
		for index := range app.children {
			if app.children[index].PID == child.PID {
				exitedAt := time.Now().UTC()
				app.children[index].ExitedAt = &exitedAt
				exitCode := 0
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						exitCode = exitErr.ExitCode()
					} else {
						exitCode = -1
					}
				}
				app.children[index].ExitCode = &exitCode
				break
			}
		}
		app.mu.Unlock()
		_ = app.persistState()
	}()

	return child, nil
}

func (app *harnessApp) currentHTTPHealthMode() healthMode {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.httpHealthMode
}

func (app *harnessApp) currentTCPHealthMode() healthMode {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.tcpHealthMode
}

func (app *harnessApp) setHTTPHealthMode(mode healthMode) error {
	app.mu.Lock()
	app.httpHealthMode = mode
	currentServer := app.httpHealthServer
	app.mu.Unlock()

	if mode == healthStopped {
		if currentServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := currentServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
		app.mu.Lock()
		app.httpHealthServer = nil
		app.mu.Unlock()
		return nil
	}

	if currentServer != nil {
		return nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+app.httpHealthPort)
	if err != nil {
		return fmt.Errorf("listen http health server: %w", err)
	}

	server := &http.Server{
		Handler:           http.HandlerFunc(app.handleDedicatedHTTPHealth),
		ReadHeaderTimeout: 5 * time.Second,
	}

	app.mu.Lock()
	app.httpHealthServer = server
	app.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.setLastError("http health server: " + err.Error())
		}
		app.mu.Lock()
		if app.httpHealthServer == server {
			app.httpHealthServer = nil
		}
		app.mu.Unlock()
	}()

	return nil
}

func (app *harnessApp) setTCPHealthMode(mode healthMode) error {
	app.mu.Lock()
	app.tcpHealthMode = mode
	currentListener := app.tcpHealthLn
	app.mu.Unlock()

	if mode == healthStopped {
		if currentListener != nil {
			if err := currentListener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				return err
			}
		}
		app.mu.Lock()
		app.tcpHealthLn = nil
		app.mu.Unlock()
		return nil
	}

	if currentListener != nil {
		return nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+app.tcpHealthPort)
	if err != nil {
		return fmt.Errorf("listen tcp health server: %w", err)
	}

	app.mu.Lock()
	app.tcpHealthLn = listener
	app.mu.Unlock()

	go app.serveTCPHealth(listener)
	return nil
}

func (app *harnessApp) handleDedicatedHTTPHealth(w http.ResponseWriter, _ *http.Request) {
	mode := app.currentHTTPHealthMode()
	status := http.StatusOK
	bodyStatus := "ok"
	if mode == healthError {
		status = http.StatusInternalServerError
		bodyStatus = "error"
	} else if mode == healthStopped {
		status = http.StatusServiceUnavailable
		bodyStatus = "stopped"
	}

	app.writeJSON(w, status, map[string]any{
		"service": app.serviceName,
		"status":  bodyStatus,
		"mode":    string(mode),
		"port":    app.httpHealthPort,
	})
}

func (app *harnessApp) serveTCPHealth(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			app.setLastError("tcp health server: " + err.Error())
			return
		}

		go func(connection net.Conn) {
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(2 * time.Second))
			mode := app.currentTCPHealthMode()
			payload := "OK\n"
			if mode == healthError {
				payload = "ERROR\n"
			}
			_, _ = io.WriteString(connection, payload)
		}(conn)
	}
}

func (app *harnessApp) closeHealthServers() {
	app.mu.Lock()
	httpServer := app.httpHealthServer
	tcpListener := app.tcpHealthLn
	app.httpHealthServer = nil
	app.tcpHealthLn = nil
	app.httpHealthMode = healthStopped
	app.tcpHealthMode = healthStopped
	app.mu.Unlock()

	if httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = httpServer.Shutdown(shutdownCtx)
		cancel()
	}
	if tcpListener != nil {
		_ = tcpListener.Close()
	}
}

func (app *harnessApp) serviceEnv() map[string]string {
	return map[string]string{
		"ECHO_MESSAGE":           app.message,
		"ECHO_PORT":              app.port,
		"ECHO_LOG_PATH":          app.logPath,
		"ECHO_STATE_PATH":        app.statePath,
		"ECHO_DB_PATH":           app.dbPath,
		"ECHO_HTTP_HEALTH_PORT":  app.httpHealthPort,
		"ECHO_TCP_PORT":          app.tcpHealthPort,
		"ECHO_HTTP_HEALTH_MODE":  string(app.currentHTTPHealthMode()),
		"ECHO_TCP_HEALTH_MODE":   string(app.currentTCPHealthMode()),
	}
}

func (app *harnessApp) serviceLassoOutput() map[string]any {
	snapshot := app.snapshot()
	serviceEnv := app.serviceEnv()
	globalEnv := cloneStringMap(app.globalEnv)

	return map[string]any{
		"serviceId": app.serviceName,
		"env":       serviceEnv,
		"globalEnv": globalEnv,
		"globalenv": globalEnv,
		"healthTargets": map[string]any{
			"process": map[string]any{
				"pid":    snapshot.PID,
				"status": "ok",
			},
			"http": map[string]any{
				"url":  snapshot.HTTPHealthURL,
				"mode": snapshot.HTTPHealthMode,
			},
			"tcp": map[string]any{
				"address": snapshot.TCPHealthAddr,
				"mode":    snapshot.TCPHealthMode,
			},
		},
	}
}

func runChild(name string, oneShot bool) error {
	logPath := envOrDefault("ECHO_LOG_PATH", "./runtime/child.log")
	statePath := envOrDefault("ECHO_STATE_PATH", "./runtime/child.state.json")
	dbPath := envOrDefault("ECHO_DB_PATH", "./runtime/child.sqlite")

	for _, candidate := range []string{logPath, statePath, dbPath} {
		if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
			return err
		}
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := createSchema(db); err != nil {
		return err
	}

	now := time.Now().UTC()
	if _, err := fmt.Fprintf(logFile, "%s [child-start] %s\n", now.Format(time.RFC3339), name); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT INTO events (kind, detail, created_at) VALUES (?, ?, ?)`, "child-start", name, now.Format(time.RFC3339)); err != nil {
		return err
	}

	state := map[string]any{
		"name":      name,
		"pid":       os.Getpid(),
		"oneShot":   oneShot,
		"startedAt": now,
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(statePath, payload, 0o644); err != nil {
		return err
	}

	if oneShot {
		return nil
	}

	ctx, stop := signalNotifyContext()
	defer stop()
	<-ctx.Done()

	endedAt := time.Now().UTC()
	if _, err := fmt.Fprintf(logFile, "%s [child-stop] %s\n", endedAt.Format(time.RFC3339), name); err != nil {
		return err
	}
	_, _ = db.Exec(`INSERT INTO events (kind, detail, created_at) VALUES (?, ?, ?)`, "child-stop", name, endedAt.Format(time.RFC3339))
	return nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			detail TEXT NOT NULL,
			created_at TEXT NOT NULL
		)
	`)
	return err
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseHealthMode(raw string) healthMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(healthHealthy):
		return healthHealthy
	case string(healthStopped), "stop":
		return healthStopped
	case string(healthError):
		return healthError
	default:
		return healthHealthy
	}
}

func resolveChildExecutable() (string, error) {
	if configured := os.Getenv("ECHO_CHILD_EXECUTABLE"); configured != "" {
		return configured, nil
	}

	return os.Executable()
}

func collectEnv() map[string]string {
	entries := map[string]string{}
	for _, raw := range os.Environ() {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) == 2 {
			entries[parts[0]] = parts[1]
		}
	}
	return entries
}

func collectGlobalEnv() map[string]string {
	collected := map[string]string{}
	currentEnv := collectEnv()

	if raw := currentEnv["SERVICE_LASSO_GLOBAL_ENV_JSON"]; raw != "" {
		var decoded map[string]any
		if json.Unmarshal([]byte(raw), &decoded) == nil {
			for key, value := range decoded {
				collected[key] = fmt.Sprint(value)
			}
		}
	}

	for key, value := range currentEnv {
		switch {
		case strings.HasPrefix(key, "SERVICE_LASSO_GLOBAL_") && key != "SERVICE_LASSO_GLOBAL_ENV_JSON":
			collected[strings.TrimPrefix(key, "SERVICE_LASSO_GLOBAL_")] = value
		case strings.HasPrefix(key, "GLOBAL_"):
			collected[strings.TrimPrefix(key, "GLOBAL_")] = value
		}
	}

	return collected
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func defaultMessage(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func signalNotifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Echo Service Harness</title>
  <style>
    body { font-family: sans-serif; margin: 2rem; background: #f5f6f8; color: #18202a; }
    h1 { margin-bottom: 0.25rem; }
    h2 { margin-top: 2rem; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.75rem; margin: 1rem 0 1.5rem; }
    button { padding: 0.8rem 1rem; border: 0; border-radius: 0.6rem; background: #1d4ed8; color: white; cursor: pointer; }
    button.secondary { background: #0f766e; }
    button.warn { background: #b45309; }
    button.danger { background: #b91c1c; }
    pre { background: white; padding: 1rem; border-radius: 0.75rem; overflow: auto; min-height: 220px; }
  </style>
</head>
<body>
  <h1>Echo Service Harness</h1>
  <p>Manual UI for driving the same API actions used by automation and future CLI checks.</p>

  <h2>Persistence</h2>
  <div class="grid">
    <button onclick="runAction('/action/write-log')">Write Log</button>
    <button onclick="runAction('/action/write-state')" class="secondary">Write State</button>
    <button onclick="runAction('/action/write-sqlite')" class="secondary">Write SQLite</button>
    <button onclick="runAction('/action/write-stdout')">Write Stdout</button>
    <button onclick="runAction('/action/write-stderr')" class="warn">Write Stderr</button>
  </div>

  <h2>Health Controls</h2>
  <div class="grid">
    <button onclick="runModeAction('/action/http-health', 'healthy')">HTTP Healthy</button>
    <button onclick="runModeAction('/action/http-health', 'error')" class="warn">HTTP Error</button>
    <button onclick="runModeAction('/action/http-health', 'stopped')" class="danger">HTTP Stopped</button>
    <button onclick="runModeAction('/action/tcp-health', 'healthy')">TCP Healthy</button>
    <button onclick="runModeAction('/action/tcp-health', 'error')" class="warn">TCP Error</button>
    <button onclick="runModeAction('/action/tcp-health', 'stopped')" class="danger">TCP Stopped</button>
    <button onclick="fetchView('/health/http')">View HTTP Health Status</button>
    <button onclick="fetchView('/health/tcp')">View TCP Health Status</button>
  </div>

  <h2>Children And State</h2>
  <div class="grid">
    <button onclick="runAction('/action/start-child')">Start Child</button>
    <button onclick="runAction('/action/fork-child')">Fork Child</button>
    <button onclick="fetchView('/state')">View State</button>
    <button onclick="fetchView('/logs')">View Logs</button>
    <button onclick="fetchView('/sqlite')">View SQLite</button>
  </div>

  <h2>Environment</h2>
  <div class="grid">
    <button onclick="fetchView('/env')">View Env</button>
    <button onclick="fetchView('/global-env')">View Global Env</button>
    <button onclick="fetchView('/service-lasso/output')">View Service Lasso Output</button>
  </div>

  <h2>Lifecycle</h2>
  <div class="grid">
    <button onclick="runAction('/action/error')" class="warn">Simulate Error</button>
    <button onclick="runAction('/action/close')" class="danger">Close</button>
    <button onclick="runAction('/action/abort')" class="danger">Abort</button>
  </div>

  <pre id="output">Ready.</pre>
  <script>
    async function runAction(path) {
      const response = await fetch(path, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ message: path + ' from UI' })
      });
      const body = await response.json();
      document.getElementById('output').textContent = JSON.stringify(body, null, 2);
    }
    async function runModeAction(path, mode) {
      const response = await fetch(path, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ mode, message: path + ':' + mode + ' from UI' })
      });
      const body = await response.json();
      document.getElementById('output').textContent = JSON.stringify(body, null, 2);
    }
    async function fetchView(path) {
      const response = await fetch(path);
      const body = await response.json();
      document.getElementById('output').textContent = JSON.stringify(body, null, 2);
    }
  </script>
</body>
</html>`
