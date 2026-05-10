# tui-tube

A terminal UI for browsing and playing YouTube music from curated channels. Built on the [Charm](https://charm.sh) stack with a local SQLite catalog.

## What it does

- Browse a searchable table of tracks pulled from curated YouTube channels
- Stream any track instantly via `mpv` (no window, background audio)
- Download tracks to `/music/yt-radio` on demand
- Favorite tracks and organize them into playlists
- Sync new uploads from channels in the background

## Stack

- [Bubble Tea v2](https://charm.land/bubbletea/v2) — TUI framework
- [Lip Gloss v2](https://charm.land/lipgloss/v2) — styling and layout
- [Bubbles](https://github.com/charmbracelet/bubbles) — table, text input
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) — pure Go SQLite driver (no CGo)
- `yt-dlp` — metadata sync and audio download
- `mpv` — headless audio playback

## Screens

| Screen | Description |
|--------|-------------|
| Library | Main track table with search. Browse all songs across all stations. |
| Stations | List of synced YouTube channels with track counts and last sync time. |
| Favorites | Tracks you've saved. Acts as a default playlist. |
| Playlists | User-created playlists. |
| Settings | Theme, download path, playback options. |

## Keybindings

| Key | Action |
|-----|--------|
| `enter` | Stream selected track via mpv |
| `d` | Download selected track to `/music/yt-radio` |
| `f` | Toggle favorite on selected track |
| `p` | Add selected track to a playlist |
| `/` | Focus search input |
| `esc` | Clear search / close overlay |
| `ctrl+k` | Command palette |
| `?` | Help overlay |
| `ctrl+t` | Cycle theme |
| `q` | Quit |

## Database

Local SQLite at `~/.local/share/tui-tube/tui-tube.db`. Schema:

```
stations      — YouTube channels being tracked
tracks        — All synced videos (youtube_id, title, artist, station)
tracks_fts    — FTS5 full-text search index over title + artist
playlists     — User-created playlists (Favorites is id=1)
playlist_tracks — Junction table linking tracks to playlists
```

The DB is fully portable — export to PostgreSQL or Convex at any time with a simple migration script. The `youtube_id` field is the natural key, so nothing is locked to SQLite-specific IDs.

## Setup

### Prerequisites

```bash
paru -S yt-dlp mpv
```

### First run — import from Convex snapshot

If migrating from a Convex export:

```bash
python3 scripts/import_convex.py \
  --snapshot ./temp/snapshot_*/  \
  --db ~/.local/share/tui-tube/tui-tube.db
```

### First run — fresh sync from channels

```bash
tui-tube sync   # pulls metadata from all configured stations (no download)
tui-tube        # launch the TUI
```

## Development

```bash
go run ./cmd/tui-tube    # run
go test ./...             # test
go build ./cmd/tui-tube  # build check
```

## Planned

- [ ] Background channel sync (goroutine, configurable interval)
- [ ] Now playing bar in footer
- [ ] mpv socket control (pause/resume/seek without leaving TUI)
- [ ] Export playlist to m3u
- [ ] PostgreSQL/Convex migration script for web version
