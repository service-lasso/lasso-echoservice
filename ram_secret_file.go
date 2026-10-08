package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ramUNC = regexp.MustCompile(`^\\\\127\.0\.0\.1@([0-9]{1,5})\\DavWWWRoot\\([a-f0-9]{64})\\?$`)
var ramCapabilityPath = regexp.MustCompile(`^/[a-f0-9]{64}/?$`)
var ramFileName = regexp.MustCompile(`^[a-zA-Z0-9_-][a-zA-Z0-9._-]{0,127}$`)

type ramFileReadStatus struct {
	Status     string `json:"status"`
	SizeBytes  int    `json:"sizeBytes"`
	Reads      int    `json:"reads"`
	LastReadAt string `json:"lastReadAt,omitempty"`
}

// Convert Core's UNC directory into the same loopback HTTP transport. This
// portable consumer does not need a drive mapping or the Windows WebClient.
func ramFileURL(directory, name string) (string, error) {
	invalid := errors.New("invalid RAM secret file location")
	if !ramFileName.MatchString(name) {
		return "", invalid
	}
	if parts := ramUNC.FindStringSubmatch(directory); parts != nil {
		directory = "http://127.0.0.1:" + parts[1] + "/" + parts[2]
	}
	u, err := url.Parse(directory)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || !ramCapabilityPath.MatchString(u.Path) {
		return "", invalid
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 || u.Host != net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) {
		return "", invalid
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + name
	return u.String(), nil
}

func readRAMSecretFile(directory, name string) (int, error) {
	target, err := ramFileURL(directory, name)
	if err != nil {
		return 0, err
	}
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return 0, errors.New("RAM secret file unavailable")
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, errors.New("RAM secret file unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, errors.New("RAM secret file unavailable")
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(content) > 256<<10 {
		return 0, errors.New("RAM secret file unavailable")
	}
	// Demonstrate an application consuming and validating its configuration.
	// The credential never enters Echo's public snapshot, log, or SQLite events.
	var config struct {
		DemoCredential string `json:"demoCredential"`
	}
	if json.Unmarshal(content, &config) != nil || config.DemoCredential == "" {
		return 0, errors.New("RAM secret file invalid")
	}
	return len(content), nil
}

func (app *harnessApp) refreshRAMFile() {
	app.ramReadMu.Lock()
	defer app.ramReadMu.Unlock()
	directory := os.Getenv("ECHO_SECRET_FILES_DIR")
	if directory == "" {
		app.mu.Lock()
		app.ramFile.Status = "disabled"
		app.mu.Unlock()
		return
	}
	size, err := readRAMSecretFile(directory, envOrDefault("ECHO_SECRET_FILE_NAME", "demo-config.json"))
	app.mu.Lock()
	defer app.mu.Unlock()
	if err != nil {
		app.ramFile.Status = "unavailable"
		app.ramFile.SizeBytes = 0
		return
	}
	app.ramFile.Status = "loaded"
	app.ramFile.SizeBytes = size
	app.ramFile.Reads++
	app.ramFile.LastReadAt = time.Now().UTC().Format(time.RFC3339Nano)
}

func (app *harnessApp) handleRAMFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodPost {
		app.refreshRAMFile()
	}
	app.mu.Lock()
	status := app.ramFile
	app.mu.Unlock()
	app.writeJSON(w, http.StatusOK, status)
}
