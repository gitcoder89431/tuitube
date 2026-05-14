# tuitube — Claude Onboarding

tuitube is a terminal music player for curated YouTube channels. It streams audio via mpv, manages a local SQLite library of 8000+ tracks, and exposes an MCP server so Claude can control playback, curate playlists, and manage the library directly.

## What Claude can do

Claude has live MCP tools connected to your library. You can ask things like:

- *"Play something chill"* — Claude searches and streams a track
- *"Make me a late night trap playlist"* — Claude searches, creates the playlist, and adds tracks
- *"Add the 7clouds Chill station and sync it"* — Claude discovers the channel and pulls all tracks
- *"What's in my claude playlist?"* — Claude lists the tracks
- *"Skip / pause / stop"* — Claude controls mpv directly

Claude does not need the TUI open to work. If Claude starts a track, the TUI header shows it automatically. The Logs screen shows recent agent activity in accent color.

## MCP Setup

Register tuitube as an MCP server once — Claude Code handles the rest:

```bash
claude mcp add tuitube tuitube mcp
```

This adds it to `~/.claude.json`. The server starts automatically when Claude Code opens. To remove: `claude mcp remove tuitube`.

> **Note:** `tuitube` must be in your PATH. If you installed via AUR or copied the binary manually, verify with `which tuitube` first.

## MCP tools available

| Tool | What it does |
|------|-------------|
| `search_tracks` | Search by artist, title, or both (limit optional) |
| `play_track` / `stop_playback` | Stream or stop via mpv |
| `toggle_favorite` | Favorite a track by ID |
| `list_playlists` / `create_playlist` | Manage playlists |
| `add_to_playlist` | Add multiple tracks at once (array of IDs) |
| `list_playlist_tracks` | See what's in a playlist |
| `list_stations` / `add_station` | View or add YouTube channels |
| `sync_station` | Pull new uploads from a channel |

## Adding a new YouTube channel

```
add_station --url "https://www.youtube.com/@ChannelName/videos" --name "Display Name"
```

Works with any URL format — `@handle`, `/channel/UC...`, `/c/`, `/user/`.

## Managing playlists via SQL

For bulk operations, direct SQL is faster than MCP:

```bash
sqlite3 ~/.local/share/tuitube/tuitube.db
```

```sql
-- list playlists
SELECT id, name FROM playlists;

-- rename a playlist
UPDATE playlists SET name = 'new name' WHERE id = 2;

-- add all tracks from an artist to a playlist
INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id)
SELECT 2, id FROM tracks WHERE artist LIKE '%Owlh%';

-- see what's in a playlist
SELECT t.song_title, t.artist
FROM playlist_tracks pt JOIN tracks t ON t.id = pt.track_id
WHERE pt.playlist_id = 2 ORDER BY pt.added_at DESC;
```

## Syncing new music

```bash
tuitube sync                        # all stations
tuitube sync --station <station_id> # one station
```

Station IDs from `list_stations` or `SELECT id, name FROM stations;`.

## Database location

Default: `~/.local/share/tuitube/tuitube.db`

Override: `tuitube --db /path/to/your.db`

Two-file model: `catalog.db` (curated tracks, ships with releases) merges into the user DB on launch. User playlists and favorites are never overwritten.

## TUI keybindings

| Key | Action |
|-----|--------|
| `enter` | Play / pause selected track |
| `n` | Next track in queue |
| `p` | Pause / resume |
| `space` | Toggle favorite |
| `f` | Filter favorites only |
| `d` | Download to `~/Music/tuitube` |
| `/` | Search (esc to clear) |
| `← →` | Seek ±5 seconds |
| `v` | Visualizer cycle (matrix → synthwave → off) |
| `s` | Toggle sidebar |
| `ctrl+t` | Cycle theme |
| `?` | Help screen |
| `q` | Quit |
