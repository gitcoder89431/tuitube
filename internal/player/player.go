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
	"syscall"
	"time"
)

const (
	SocketPath = "/tmp/tuitube-mpv.sock"
	StatePath  = "/tmp/tuitube-now-playing.json"
	PIDPath    = "/tmp/tuitube-mpv.pid"
)

type State struct {
	YoutubeID string `json:"youtube_id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Playing   bool   `json:"playing"`
	Paused    bool   `json:"paused"`
	Finished  bool   `json:"finished"` // true when mpv exited naturally (track ended)
}

// Play starts or replaces the current track. Kills any existing mpv first.
func Play(youtubeID, title, artist string) error {
	Stop()

	url := "https://www.youtube.com/watch?v=" + youtubeID
	cmd := exec.Command("mpv",
		"--no-video",
		"--really-quiet",
		"--gapless-audio=yes",
		"--input-ipc-server="+SocketPath,
		url,
	)
	if err := cmd.Start(); err != nil {
		return err
	}

	_ = os.WriteFile(PIDPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	go func() {
		cmd.Wait()
		// Only mark finished if we weren't manually stopped (PID file still exists)
		if _, err := os.Stat(PIDPath); err == nil {
			_ = os.Remove(PIDPath)
			_ = os.Remove(SocketPath)
			s := readState()
			s.Playing = false
			s.Finished = true
			_ = writeState(s)
		}
	}()

	return writeState(State{
		YoutubeID: youtubeID,
		Title:     title,
		Artist:    artist,
		Playing:   true,
	})
}

// Stop kills the current mpv process. Does NOT set Finished — caller stopped it.
func Stop() {
	if sendIPC(`{"command":["quit"]}`) == nil {
		time.Sleep(100 * time.Millisecond)
	}
	if data, err := os.ReadFile(PIDPath); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		}
	}
	_ = os.Remove(PIDPath)
	_ = os.Remove(SocketPath)
	_ = writeState(State{Playing: false, Finished: false})
}

// TogglePause sends a pause cycle command to mpv and updates state.
func TogglePause() error {
	if err := sendIPC(`{"command":["cycle","pause"]}`); err != nil {
		return err
	}
	s := readState()
	s.Paused = !s.Paused
	return writeState(s)
}

// NowPlaying returns current state. Returns nil if nothing is playing or finishing.
func NowPlaying() *State {
	s := readState()
	if !s.Playing && !s.Finished {
		return nil
	}
	return &s
}

// IsAlive returns true if the mpv process from the PID file is still running.
func IsAlive() bool {
	data, err := os.ReadFile(PIDPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func readState() State {
	data, err := os.ReadFile(StatePath)
	if err != nil {
		return State{}
	}
	var s State
	_ = json.Unmarshal(data, &s)
	return s
}

func writeState(s State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath, data, 0644)
}

// SeekForward seeks 5 seconds forward.
func SeekForward() error {
	return sendIPC(`{"command":["seek",5]}`)
}

// SeekBackward seeks 5 seconds backward.
func SeekBackward() error {
	return sendIPC(`{"command":["seek",-5]}`)
}

// Progress returns the current playback position and total duration in seconds.
// Returns zeros if mpv isn't running or the query fails.
func Progress() (timePos, duration float64) {
	timePos, _ = queryIPCFloat(`{"command":["get_property","time-pos"]}`)
	duration, _ = queryIPCFloat(`{"command":["get_property","duration"]}`)
	return
}

func sendIPC(cmd string) error {
	conn, err := net.DialTimeout("unix", SocketPath, 200*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd + "\n"))
	return err
}

func queryIPCFloat(cmd string) (float64, error) {
	conn, err := net.DialTimeout("unix", SocketPath, 200*time.Millisecond)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return 0, err
	}
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return 0, err
	}
	var val float64
	json.Unmarshal(resp.Data, &val)
	return val, nil
}
