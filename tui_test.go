package main

import (
	"strings"
	"testing"
	"time"
)

func sampleSnapshot() harnessSnapshot {
	return harnessSnapshot{
		Service:        "echo-service",
		PID:            4242,
		Message:        "hello from echo-service harness",
		StartedAt:      time.Now().Add(-90 * time.Second),
		LastAction:     "startup",
		ActionCount:    3,
		HTTPHealthURL:  "http://127.0.0.1:4011/health",
		HTTPHealthMode: "healthy",
		TCPHealthAddr:  "127.0.0.1:4012",
		TCPHealthMode:  "healthy",
		RecentEvents: []eventRow{
			{Kind: "tui-echo", Detail: "hello from echo-service harness", CreatedAt: time.Now()},
			{Kind: "startup", Detail: "hello from echo-service harness", CreatedAt: time.Now()},
		},
	}
}

func TestRenderDashboardFillsTheWindow(t *testing.T) {
	frame := renderDashboard(sampleSnapshot(), 80, 24)
	lines := strings.Split(frame, "\r\n")
	if len(lines) != 24 {
		t.Fatalf("lines = %d, want 24", len(lines))
	}
	for index, line := range lines {
		if len([]rune(line)) != 80 {
			t.Fatalf("line %d width = %d, want 80: %q", index, len([]rune(line)), line)
		}
	}
	for _, want := range []string{"Echo Service TUI", "pid 4242", "hello from echo-service harness", "recent events", "tui-echo", "q quit"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("frame missing %q", want)
		}
	}
}

func TestRenderDashboardClampsSmallWindows(t *testing.T) {
	frame := renderDashboard(sampleSnapshot(), 10, 3)
	lines := strings.Split(frame, "\r\n")
	for index, line := range lines {
		if width := len([]rune(line)); width != 40 {
			t.Fatalf("line %d width = %d, want 40", index, width)
		}
	}
	if len(lines) > 10 {
		t.Fatalf("lines = %d, want at most 10", len(lines))
	}
}

func TestRenderDashboardStripsControlSequences(t *testing.T) {
	snapshot := sampleSnapshot()
	snapshot.Message = "hello\x1b]0;pwned\x07\r\nsecond line"
	snapshot.RecentEvents[0].Detail = "\x1b[31mred\x1b[0m"
	frame := renderDashboard(snapshot, 80, 24)
	if strings.Contains(frame, "\x1b") || strings.Contains(frame, "\x07") {
		t.Fatalf("frame retained terminal control bytes: %q", frame)
	}
	if !strings.Contains(frame, "hello]0;pwnedsecond line") {
		t.Fatalf("frame did not reduce the value to printable text: %q", frame)
	}
}
