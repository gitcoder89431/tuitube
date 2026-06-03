// Package player manages a single shared mpv process across TUI and MCP.
// State is persisted to a JSON file so any process can read what's playing.
package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Config holds the filesystem paths used by a Player instance.
// All processes (TUI, MCP server, status command) must use the same paths to
// share state. Use DefaultConfig() for the standard XDG locations.
type Config struct {
	SocketPath string
	StatePath  string
	PIDPath    string
}

// DefaultConfig returns a Config backed by $XDG_RUNTIME_DIR (Linux) or
// os.TempDir() on other platforms.
func DefaultConfig() Config {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return Config{
		SocketPath: filepath.Join(dir, "tuitube-mpv.sock"),
		StatePath:  filepath.Join(dir, "tuitube-now-playing.json"),
		PIDPath:    filepath.Join(dir, "tuitube-mpv.pid"),
	}
}

// Player controls a single mpv process and persists playback state to disk.
type Player struct {
	cfg Config
}

// New creates a Player with the given config.
func New(cfg Config) *Player {
	return &Player{cfg: cfg}
}

type State struct {
	YoutubeID string  `json:"youtube_id"`
	Title     string  `json:"title"`
	Artist    string  `json:"artist"`
	Playing   bool    `json:"playing"`
	Paused    bool    `json:"paused"`
	Finished  bool    `json:"finished"`   // true when mpv exited naturally (track ended)
	ResumePos float64 `json:"resume_pos"` // seconds to resume from on next play; 0 = start
}

// Play starts or replaces the current track. Kills any existing mpv first.
// If localPath is non-empty and the file exists, it is played directly;
// otherwise mpv streams from YouTube.
// startPos > 0 seeks to that position in seconds before playback begins.
func (p *Player) Play(youtubeID, title, artist, localPath string, startPos float64) error {
	p.Stop()

	source := "https://www.youtube.com/watch?v=" + youtubeID
	if localPath != "" {
		if _, err := os.Stat(localPath); err == nil {
			source = localPath
		}
	}
	args := []string{
		"--no-video",
		"--really-quiet",
		"--gapless-audio=yes",
		"--input-ipc-server=" + p.cfg.SocketPath,
	}
	if startPos > 0 {
		args = append(args, fmt.Sprintf("--start=%.1f", startPos))
	}
	args = append(args, source)
	cmd := exec.Command("mpv", args...)
	if err := cmd.Start(); err != nil {
		return err
	}

	pid := cmd.Process.Pid
	_ = os.WriteFile(p.cfg.PIDPath, []byte(strconv.Itoa(pid)), 0644)

	go func() {
		cmd.Wait()
		// Only mark finished if the PID file still points to our process,
		// not a newer track that started before we exited.
		if data, err := os.ReadFile(p.cfg.PIDPath); err == nil {
			if strings.TrimSpace(string(data)) == strconv.Itoa(pid) {
				_ = os.Remove(p.cfg.PIDPath)
				_ = os.Remove(p.cfg.SocketPath)
				s := p.readState()
				s.Playing = false
				s.Finished = true
				_ = p.writeState(s)
			}
		}
	}()

	return p.writeState(State{
		YoutubeID: youtubeID,
		Title:     title,
		Artist:    artist,
		Playing:   true,
		// ResumePos intentionally zero — cleared on fresh play
	})
}

// Stop kills the current mpv process. Does NOT set Finished — caller stopped it.
func (p *Player) Stop() {
	if p.sendIPC(`{"command":["quit"]}`) == nil {
		time.Sleep(100 * time.Millisecond)
	}
	if data, err := os.ReadFile(p.cfg.PIDPath); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
			}
		}
	}
	_ = os.Remove(p.cfg.PIDPath)
	_ = os.Remove(p.cfg.SocketPath)
	_ = p.writeState(State{Playing: false, Finished: false})
}

// TogglePause sends a pause cycle command to mpv and updates state.
func (p *Player) TogglePause() error {
	if err := p.sendIPC(`{"command":["cycle","pause"]}`); err != nil {
		return err
	}
	paused, err := p.queryIPCBool(`{"command":["get_property","pause"]}`)
	if err != nil {
		return err
	}
	s := p.readState()
	s.Paused = paused
	return p.writeState(s)
}

// NowPlaying returns current state. Returns nil if nothing is playing or finishing.
func (p *Player) NowPlaying() *State {
	s := p.readState()
	if !s.Playing && !s.Finished {
		return nil
	}
	return &s
}

// IsAlive returns true if the mpv process from the PID file is still running.
func (p *Player) IsAlive() bool {
	data, err := os.ReadFile(p.cfg.PIDPath)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func (p *Player) readState() State {
	data, err := os.ReadFile(p.cfg.StatePath)
	if err != nil {
		return State{}
	}
	var s State
	_ = json.Unmarshal(data, &s)
	return s
}

func (p *Player) writeState(s State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(p.cfg.StatePath, data, 0644)
}

// SavePosition reads the current playback position and persists it to state
// so session resume can restart from the same point.
func (p *Player) SavePosition() {
	tp, _ := p.queryIPCFloat(`{"command":["get_property","time-pos"]}`)
	if tp <= 0 {
		return
	}
	s := p.readState()
	s.ResumePos = tp
	_ = p.writeState(s)
}

// SeekForward seeks 5 seconds forward.
func (p *Player) SeekForward() error {
	return p.sendIPC(`{"command":["seek",5]}`)
}

// SeekBackward seeks 5 seconds backward.
func (p *Player) SeekBackward() error {
	return p.sendIPC(`{"command":["seek",-5]}`)
}

// Progress returns the current playback position and total duration in seconds.
// Returns zeros if mpv isn't running or the query fails.
func (p *Player) Progress() (timePos, duration float64) {
	timePos, _ = p.queryIPCFloat(`{"command":["get_property","time-pos"]}`)
	duration, _ = p.queryIPCFloat(`{"command":["get_property","duration"]}`)
	return
}

// dialIPC connects to the mpv IPC socket, retrying for up to timeout.
// mpv needs ~100ms after launch before the socket is ready; this prevents
// the first Progress() or query after Play() from failing silently.
func (p *Player) dialIPC(timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("unix", p.cfg.SocketPath, 50*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (p *Player) sendIPC(cmd string) error {
	conn, err := p.dialIPC(200 * time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(cmd + "\n"))
	return err
}

func (p *Player) queryIPC(cmd string) (json.RawMessage, error) {
	conn, err := p.dialIPC(1 * time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return nil, err
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}
	if resp.Error != "" && resp.Error != "success" {
		return nil, fmt.Errorf("mpv ipc error: %s", resp.Error)
	}
	return resp.Data, nil
}

func (p *Player) queryIPCFloat(cmd string) (float64, error) {
	data, err := p.queryIPC(cmd)
	if err != nil {
		return 0, err
	}
	var val float64
	if err := json.Unmarshal(data, &val); err != nil {
		return 0, err
	}
	return val, nil
}

func (p *Player) queryIPCBool(cmd string) (bool, error) {
	data, err := p.queryIPC(cmd)
	if err != nil {
		return false, err
	}
	var val bool
	if err := json.Unmarshal(data, &val); err != nil {
		return false, err
	}
	return val, nil
}
