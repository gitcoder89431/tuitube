// Package agentlog provides a shared log file that both the MCP server and
// TUI can read, so agent actions (playlist creation, playback, search) are
// visible in the Logs screen.
package agentlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Time    time.Time `json:"t"`
	Message string    `json:"msg"`
}

func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/tuitube-agent.log"
	}
	return filepath.Join(home, ".local", "share", "tuitube", "agent.log")
}

// Write appends a single agent log entry.
// Clear wipes the agent log — call on TUI startup so each session is fresh.
func Clear() {
	_ = os.Remove(Path())
}

func Write(message string) {
	p := Path()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(Entry{Time: time.Now(), Message: message})
	f.Write(append(b, '\n'))
}

// Read returns all entries from the agent log file, newest first.
func Read() []Entry {
	data, err := os.ReadFile(Path())
	if err != nil {
		return nil
	}
	var entries []Entry
	for _, line := range splitLines(data) {
		var e Entry
		if json.Unmarshal(line, &e) == nil {
			entries = append(entries, e)
		}
	}
	// reverse: newest first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	return lines
}
