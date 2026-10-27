package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const tuiFooter = " e echo   r refresh   q quit "

// runTUI renders a full-screen dashboard while the service keeps serving its
// normal HTTP, health and FastCGI endpoints. It is a test surface for PTY
// attachment: it requires a real terminal, so a non-TTY launch simply skips it.
func (app *harnessApp) runTUI() {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		log.Printf("echo-service tui unavailable: %v", err)
		return
	}
	defer func() { _ = term.Restore(fd, state) }()

	out := os.Stdout
	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")

	keys := make(chan byte, 16)
	go func() {
		defer close(keys)
		buffer := make([]byte, 1)
		for {
			count, readErr := os.Stdin.Read(buffer)
			if readErr != nil {
				return
			}
			if count == 1 {
				select {
				case keys <- buffer[0]:
				default:
				}
			}
		}
	}()

	render := func() {
		width, height, err := term.GetSize(fd)
		if err != nil || width <= 0 {
			width = 80
		}
		if err != nil || height <= 0 {
			height = 24
		}
		fmt.Fprintf(out, "\x1b[H%s", renderDashboard(app.snapshot(), width, height))
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	render()

	for {
		select {
		case key, ok := <-keys:
			if !ok {
				return
			}
			switch key {
			case 'q', 3:
				_ = app.recordEvent("tui-shutdown", "operator quit")
				app.stopCh <- "tui-shutdown"
				return
			case 'e':
				_ = app.recordEvent("tui-echo", app.message)
				render()
			case 'r':
				render()
			}
		case <-ticker.C:
			render()
		}
	}
}

// renderDashboard builds one full frame. Every field is reduced to printable
// text first so a log value cannot inject terminal control sequences into the
// operator's emulator.
func renderDashboard(snapshot harnessSnapshot, width, height int) string {
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}
	inner := width - 4
	border := "+" + strings.Repeat("-", width-2) + "+"
	line := func(content string) string {
		return "| " + padTo(fit(content, inner), inner) + " |"
	}

	uptime := time.Since(snapshot.StartedAt).Truncate(time.Second)
	body := []string{
		border,
		line(fmt.Sprintf("Echo Service TUI  pid %d  uptime %s", snapshot.PID, uptime)),
		border,
		line("message: " + snapshot.Message),
		line(fmt.Sprintf("http:    %s (%s)", snapshot.HTTPHealthURL, snapshot.HTTPHealthMode)),
		line(fmt.Sprintf("tcp:     %s (%s)", snapshot.TCPHealthAddr, snapshot.TCPHealthMode)),
		line(fmt.Sprintf("last:    %s   actions: %d", snapshot.LastAction, snapshot.ActionCount)),
	}
	if snapshot.LastError != "" {
		body = append(body, line("error:   "+snapshot.LastError))
	}
	body = append(body, border, line("recent events"))

	footer := []string{border, line(tuiFooter), border}
	eventSlots := height - len(body) - len(footer)
	if eventSlots < 1 {
		eventSlots = 1
	}
	for index := 0; index < eventSlots && index < len(snapshot.RecentEvents); index++ {
		event := snapshot.RecentEvents[index]
		body = append(body, line(fmt.Sprintf("%s [%s] %s",
			event.CreatedAt.Local().Format("15:04:05"), event.Kind, event.Detail)))
	}
	if limit := height - len(footer); len(body) > limit {
		body = body[:limit]
	}
	for len(body) < height-len(footer) {
		body = append(body, line(""))
	}

	body = append(body, footer...)
	return strings.Join(body, "\r\n")
}

// fit reduces a value to printable text and truncates it to width cells.
func fit(value string, width int) string {
	cleaned := make([]rune, 0, len(value))
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			continue
		}
		cleaned = append(cleaned, character)
	}
	if len(cleaned) > width {
		cleaned = cleaned[:width]
	}
	return string(cleaned)
}

func padTo(value string, width int) string {
	count := len([]rune(value))
	if count >= width {
		return value
	}
	return value + strings.Repeat(" ", width-count)
}
