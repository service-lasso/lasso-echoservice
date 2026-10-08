package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRAMConsumerReadsWithoutExposingOrPersistingValues(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	token := strings.Repeat("a", 64)
	value := `{"demoCredential":"synthetic-private-demo"}`
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/"+token+"/demo-config.json" {
			t.Error("wrong app file path")
		}
		_, _ = w.Write([]byte(value))
	}))
	defer server.Close()
	directory := server.URL + "/" + token
	app := newTestHarnessAppWithEnv(t, map[string]string{"ECHO_SECRET_FILES_DIR": directory})
	if app.ramFile.Status != "loaded" || app.ramFile.SizeBytes != len(value) || requests != 1 {
		t.Fatal("startup did not consume config")
	}
	for _, method := range []string{"GET", "POST"} {
		response := httptest.NewRecorder()
		app.handleRAMFile(response, httptest.NewRequest(method, "/secret-file", nil))
		if response.Code != 200 || strings.Contains(response.Body.String(), token) || strings.Contains(response.Body.String(), "synthetic-private-demo") {
			t.Fatal("consumer metadata exposed private data")
		}
	}
	if requests != 2 || app.ramFile.Reads != 2 {
		t.Fatal("GET triggered reads or POST did not refresh")
	}
	envResponse := httptest.NewRecorder()
	app.handleEnv(envResponse, httptest.NewRequest("GET", "/env", nil))
	if strings.Contains(envResponse.Body.String(), token) {
		t.Fatal("environment route exposed capability")
	}
	if err := app.persistState(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{app.statePath, app.logPath} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), token) || strings.Contains(string(content), "synthetic-private-demo") {
			t.Fatal("RAM data persisted")
		}
	}
	raw, _ := json.Marshal(app.snapshot())
	if strings.Contains(string(raw), directory) {
		t.Fatal("snapshot includes RAM path")
	}
}

func TestRAMConsumerRejectsUnsafeLocationsAndReads(t *testing.T) {
	token := strings.Repeat("a", 64)
	for _, directory := range []string{"http://evil.example:8080/" + token, "https://127.0.0.1:8080/" + token, "http://127.0.0.1:8080/" + token + "?x=1", "http://user:pass@127.0.0.1:8080/" + token, "http://127.0.0.1:8080/%61" + token[1:], "http://127.0.0.1:8080/" + token + "#private", "/tmp/files", `\\evil.example\DavWWWRoot\` + token} {
		if _, err := ramFileURL(directory, "demo-config.json"); err == nil {
			t.Fatalf("unsafe location accepted: %s", directory)
		}
	}
	for _, name := range []string{"../key", "a/b", "a\\b", ".", "a?b"} {
		if _, err := ramFileURL("http://127.0.0.1:8080/"+token, name); err == nil {
			t.Fatal("unsafe filename accepted")
		}
	}
	unc := `\\127.0.0.1@8080\DavWWWRoot\` + token
	if result, err := ramFileURL(unc, "demo-config.json"); err != nil || result != "http://127.0.0.1:8080/"+token+"/demo-config.json" {
		t.Fatal("UNC conversion failed")
	}
	for _, mode := range []string{"denied", "redirect", "oversized", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch mode {
				case "denied":
					w.WriteHeader(403)
				case "redirect":
					http.Redirect(w, r, "http://evil.example/private", 302)
				case "oversized":
					_, _ = w.Write([]byte(strings.Repeat("x", (256<<10)+1)))
				case "invalid":
					_, _ = w.Write([]byte(`{"demoCredential":""}`))
				}
			}))
			defer server.Close()
			if _, err := readRAMSecretFile(server.URL+"/"+token, "demo-config.json"); err == nil {
				t.Fatal("bad response accepted")
			}
			app := newTestHarnessAppWithEnv(t, map[string]string{"ECHO_SECRET_FILES_DIR": server.URL + "/" + token})
			if app.ramFile.Status != "unavailable" || app.ramFile.Reads != 0 {
				t.Fatal("failed read reported loaded")
			}
		})
	}
}
