package player

import (
	"path/filepath"
	"testing"
)

func testPlayer(t *testing.T) *Player {
	t.Helper()
	dir := t.TempDir()
	return New(Config{
		SocketPath: filepath.Join(dir, "mpv.sock"),
		StatePath:  filepath.Join(dir, "state.json"),
		PIDPath:    filepath.Join(dir, "mpv.pid"),
	})
}

func TestWriteReadState(t *testing.T) {
	p := testPlayer(t)

	want := State{
		YoutubeID: "abc123",
		Title:     "Test Track",
		Artist:    "Test Artist",
		Playing:   true,
		Paused:    false,
		ResumePos: 42.5,
	}
	if err := p.writeState(want); err != nil {
		t.Fatalf("writeState: %v", err)
	}

	got := p.readState()
	if got != want {
		t.Errorf("readState mismatch:\n got  %+v\n want %+v", got, want)
	}
}

func TestReadStateMissingFile(t *testing.T) {
	p := testPlayer(t)
	s := p.readState()
	if s.Playing || s.YoutubeID != "" {
		t.Errorf("expected zero state for missing file, got %+v", s)
	}
}

func TestNowPlayingNilWhenIdle(t *testing.T) {
	p := testPlayer(t)
	// no state file written — nothing playing
	if p.NowPlaying() != nil {
		t.Error("expected nil NowPlaying when no state exists")
	}
}

func TestNowPlayingReturnsStateWhenPlaying(t *testing.T) {
	p := testPlayer(t)
	_ = p.writeState(State{YoutubeID: "xyz", Playing: true})
	s := p.NowPlaying()
	if s == nil || s.YoutubeID != "xyz" {
		t.Errorf("expected playing state, got %v", s)
	}
}

func TestNowPlayingReturnsStateWhenFinished(t *testing.T) {
	p := testPlayer(t)
	_ = p.writeState(State{YoutubeID: "xyz", Finished: true})
	s := p.NowPlaying()
	if s == nil || s.YoutubeID != "xyz" {
		t.Errorf("expected finished state, got %v", s)
	}
}

func TestIsAliveReturnsFalseWhenNoPIDFile(t *testing.T) {
	p := testPlayer(t)
	if p.IsAlive() {
		t.Error("expected IsAlive=false when no PID file")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SocketPath == "" || cfg.StatePath == "" || cfg.PIDPath == "" {
		t.Errorf("DefaultConfig returned empty path: %+v", cfg)
	}
}
