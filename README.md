# tuitube

A terminal music player for curated YouTube channels. Streams via mpv, manages a local SQLite library, and exposes an MCP server so Claude can control playback and curate playlists.

## Features

- **8000+ tracks** across curated YouTube music channels — lofi, chill, trap, hip-hop, pop
- **Stream instantly** via mpv — no downloads required, background audio
- **Download** tracks to `~/Music/tuitube` on demand
- **Favorites & playlists** — space to favorite, Claude can curate playlists via MCP
- **Autoplay queue** — enter on a track builds a queue from the current view
- **Seek, pause, next** — full playback control from the keyboard
- **Matrix & synthwave visualizers** — press `v` to cycle
- **14 themes** — cycle with `ctrl+t`, live preview in the command palette
- **Claude MCP server** — play, search, create playlists, sync channels without opening the TUI

## Stack

- [Bubble Tea v2](https://charm.land/bubbletea/v2) — TUI framework
- [Lip Gloss v2](https://charm.land/lipgloss/v2) — styling and layout
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — pure Go SQLite, no CGo
- `mpv` — headless audio playback with IPC socket
- `yt-dlp` — channel sync and audio downloads

## Screens

| Screen | Description |
|--------|-------------|
| Library | Searchable track table. Browse 8000+ songs, filter favorites, open playlists. |
| Playlists | Lobby — Favorites, user playlists, and stations. Enter to browse any collection. |
| Settings | Active theme and build info. |
| Help | Full keybinding reference. |
| Logs | App events + Claude agent activity log. |

## Keybindings

| Key | Action |
|-----|--------|
| `enter` | Play selected track (pause/resume if already playing) |
| `n` | Next track in queue |
| `p` | Pause / resume |
| `space` | Toggle favorite |
| `f` | Filter to favorites only |
| `d` | Download to `~/Music/tuitube` |
| `/` | Search — live FTS, esc to clear |
| `← →` | Seek ±5 seconds |
| `v` | Cycle visualizer (matrix → synthwave → off) |
| `s` | Toggle sidebar |
| `ctrl+t` | Cycle theme |
| `ctrl+k` | Command palette |
| `?` | Help screen |
| `q` | Quit |

## Setup

### Prerequisites

```bash
paru -S mpv yt-dlp        # Arch / CachyOS
brew install mpv yt-dlp   # macOS
```

### Bootstrap with pre-seeded catalog

```bash
git clone https://github.com/gitcoder89431/tui-tube
cd tui-tube
tuitube bootstrap          # merges catalog.db → ~/.local/share/tuitube/tuitube.db
tuitube                    # launch
```

### Add a new channel

```bash
tuitube add-station --url "https://www.youtube.com/@ChannelName/videos" --name "Display Name"
tuitube sync
```

### Sync new uploads

```bash
tuitube sync               # all stations
tuitube sync --station ID  # one station
```

## Claude MCP

Register the MCP server in `~/.claude.json`:

```json
{
  "mcpServers": {
    "tuitube": {
      "command": "/path/to/tuitube",
      "args": ["--db", "~/.local/share/tuitube/tuitube.db", "mcp"]
    }
  }
}
```

Then ask Claude to play songs, create playlists, or sync channels — no TUI required.

## Database

Two-file model:

| File | Contents | Managed by |
|------|----------|------------|
| `catalog.db` | Stations + tracks (8000+) | You — ship in releases |
| `~/.local/share/tuitube/tuitube.db` | User playlists, favorites, downloads | Never overwritten on upgrade |

On launch, tuitube auto-merges a newer `catalog.db` into the user DB. New tracks appear, nothing the user added is touched.

## Development

```bash
go run ./cmd/tuitube       # run with local DB
go test ./...              # tests
go build ./cmd/tuitube     # build check
./scripts/export_catalog.sh  # export catalog.db for a release
```
