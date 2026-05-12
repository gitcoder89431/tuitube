// Package player manages a single shared mpv process across TUI and MCP.
// State is persisted to a JSON file so any process can read what's playing.
package player

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	SocketPath = "/tmp/tui-tube-mpv.sock"
	StatePath  = "/tmp/tui-tube-now-playing.json"
	PIDPath    = "/tmp/tui-tube-mpv.pid"
)

type State struct {
	YoutubeID string `json:"youtube_id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Playing   bool   `json:"playing"`
	Paused    bool   `json:"paused"`
}

// Play starts or replaces the current track. Kills any existing mpv first.
func Play(youtubeID, title, artist string) error {
	Stop()

	url := "https://www.youtube.com/watch?v=" + youtubeID
	cmd := exec.Command("mpv",
		"--no-video",
		"--really-quiet",
		"--input-ipc-server="+SocketPath,
		url,
	)
	if err := cmd.Start(); err != nil {
		return err
	}

	// write PID so any process can kill it later
	_ = os.WriteFile(PIDPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	// reap the process in the background so it doesn't become a zombie
	go cmd.Wait()

	return writeState(State{
		YoutubeID: youtubeID,
		Title:     title,
		Artist:    artist,
		Playing:   true,
	})
}

// Stop kills the current mpv process if one is running.
func Stop() {
	// try graceful quit via IPC first
	if sendIPC(`{"command":["quit"]}`) == nil {
		time.Sleep(100 * time.Millisecond)
	}

	// fallback: kill by PID
	if data, err := os.ReadFile(PIDPath); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		}
	}

	_ = os.Remove(PIDPath)
	_ = os.Remove(SocketPath)
	_ = writeState(State{Playing: false})
}

// TogglePause sends a pause cycle command to mpv and flips the paused state in the state file.
func TogglePause() error {
	if err := sendIPC(`{"command":["cycle","pause"]}`); err != nil {
		return err
	}
	data, err := os.ReadFile(StatePath)
	if err != nil {
		return err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	s.Paused = !s.Paused
	return writeState(s)
}

// NowPlaying returns the current playback state. Returns nil if nothing is playing.
func NowPlaying() *State {
	data, err := os.ReadFile(StatePath)
	if err != nil {
		return nil
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	if !s.Playing {
		return nil
	}
	return &s
}

func writeState(s State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath, data, 0644)
}

// sendIPC sends a raw JSON command to the mpv IPC socket.
func sendIPC(cmd string) error {
	conn, err := net.DialTimeout("unix", SocketPath, 200*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd + "\n"))
	return err
}
