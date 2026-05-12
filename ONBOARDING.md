# tui-tube — Claude Onboarding

tui-tube is a terminal music player for YouTube music channels. It streams audio via mpv, manages a local SQLite library of ~8000+ tracks, and exposes an MCP server so Claude can control playback, curate playlists, and manage the library directly.

## What Claude can do

Claude has live MCP tools connected to your library. You can ask things like:

- *"Play something chill"* — Claude searches the library and streams a track
- *"Make me a late night trap playlist"* — Claude searches, creates the playlist, and adds tracks in one go
- *"Add the 7clouds Chill station and sync it"* — Claude discovers the channel and pulls all tracks
- *"What's in my claude playlist?"* — Claude lists the tracks
- *"Pause / stop the music"* — Claude controls mpv directly

Claude does not need the TUI open to work. The TUI and Claude share the same playback process — if Claude starts a track, the TUI shows it in the header automatically.

## MCP tools available

| Tool | What it does |
|------|-------------|
| `search_tracks` | Search by artist, title, or both |
| `play_track` / `stop_playback` | Stream or stop via mpv |
| `toggle_favorite` | Favorite a track |
| `list_playlists` / `create_playlist` | Manage playlists |
| `add_to_playlist` | Add multiple tracks at once (pass an array of IDs) |
| `list_playlist_tracks` | See what's in a playlist |
| `list_stations` / `add_station` | View or add YouTube channels |
| `sync_station` | Pull new uploads from a channel |

## Adding a new YouTube channel

```
add_station --url "https://www.youtube.com/@ChannelName/videos" --name "Display Name"
```

Works with any YouTube URL format (`@handle`, `/channel/UC...`, `/c/`, `/user/`). After adding, sync it to pull all tracks.

## Managing playlists via SQL

For bulk operations, direct SQL is faster than MCP:

```bash
sqlite3 ~/.local/share/tui-tube/tui-tube.db
```

```sql
-- list playlists
SELECT id, name FROM playlists;

-- rename a playlist
UPDATE playlists SET name = 'new name' WHERE id = 2;

-- add tracks in bulk
INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id)
SELECT 2, id FROM tracks WHERE artist LIKE '%Owlh%';

-- remove a track from a playlist
DELETE FROM playlist_tracks WHERE playlist_id = 2 AND track_id = 'xxx';

-- see what's in a playlist
SELECT t.song_title, t.artist
FROM playlist_tracks pt JOIN tracks t ON t.id = pt.track_id
WHERE pt.playlist_id = 2 ORDER BY pt.added_at DESC;
```

## Syncing new music

```bash
# sync all stations (pulls new uploads since last sync)
tui-tube sync

# sync a specific station
tui-tube sync --station <station_id>
```

Get station IDs from `list_stations` or `SELECT id, name FROM stations;`.

## Cleaning up track titles

If you notice messy titles (extra spaces, symbols, emoji), run:

```bash
tui-tube clean
```

This applies the title normaliser to all existing tracks and rebuilds the search index.

## Database location

Default: `~/.local/share/tui-tube/tui-tube.db`

Override at runtime: `tui-tube --db /path/to/your.db`

The DB is a standard SQLite file — fully portable, no lock-in.

## TUI keybindings

| Key | Action |
|-----|--------|
| `enter` | Play selected track (or pause if already playing) |
| `p` | Pause / resume |
| `d` | Download track to `/music/yt-radio` |
| `space` | Toggle favorite |
| `f` | Filter to favorites only |
| `/` | Search (live, esc to clear) |
| `esc` | Clear search / back to playlists |
| `n` | New playlist (on Playlists screen) |
| `q` | Quit |
