# VibeFM — tui-tube Reference

## tui-tube's role

tui-tube is the **manual backend / content pipeline** for VibeFM. It owns the music catalog and all editorial content. The VibeFM web app consumes this data — it does not produce it.

Think of it as two separate systems:

| System | Role |
|--------|------|
| **tui-tube** | Playlist builder, sync tool, content authoring. Runs locally. Agent-writable. Never touches Convex directly. |
| **VibeFM web** | Consumes tui-tube's output via a seed script. Serves users. Writes user interactions (likes, dead links) back to Convex. |

When new playlists are added or content is updated in tui-tube, the operator runs the seed script in the VibeFM repo to push changes to Convex. That's the only bridge between the two systems.

---

## What lives here

| File | Purpose |
|------|---------|
| `catalog.db` | Music catalog — tracks, playlists, YouTube IDs |
| `vibefm-editorial.json` | Editorial content — liner notes, descriptions, facts, curators |

These two things are intentionally separate. `catalog.db` is managed by tui-tube's sync tools. `vibefm-editorial.json` is managed by agents and humans writing content.

---

## Data flow

```
tui-tube/catalog.db              (track catalog — synced from YouTube)
tui-tube/vibefm-editorial.json   (editorial content — written by agents)
        ↓
seed script (TODO — lives in vibefm repo /scripts)
        ↓
Convex playlists table           (editorial + metadata)
Convex tracks table              (real tracks with youtube_id — TODO, not seeded yet)
        ↓
App queries Convex at runtime
User interactions write to Convex (likes, dead link reports, user playlists)
```

**Current state:** `lib/vibefm.ts` in the vibefm repo still has a fake seeded TRACK_POOL. Real tracks from `catalog.db` are not in Convex yet. The `playlists` table exists in Convex schema but is not seeded. Seed script does not exist yet.

---

## Convex schema status

Full schema in `nextjs-chat-template/convex/schema.ts`.

| Table | Status | Notes |
|-------|--------|-------|
| `playlists` | ✅ Schema exists | `sortOrder`, `name`, `description`, `linerNotes`, `facts`, `tags`, `tracks` (count), `curatorName`, `curatorTitle`, `coverArt*`. Not seeded yet. |
| `tracks` | ✅ Schema exists | `youtubeId`, `title`, `artist`, `thumbnail` (optional), `playlistId`. Not seeded yet. |
| `userLikes` | ✅ Schema exists | `ownerTokenIdentifier`, `trackId`, `likedAt`. Note: camelCase table name. |
| `deadLinks` | ✅ Schema exists | `youtubeId`, `trackId`, `reportedBy` (optional), `reportedAt`. Note: camelCase table name. |
| `users` | ✅ Exists | Auth + profile |

## What needs building

1. **`vibefm-editorial.json`** — write curator names/titles, liner notes, and facts for all 49 playlists
2. **Seed script** in vibefm repo `/scripts` — reads `tui-tube/catalog.db` + `tui-tube/vibefm-editorial.json`, pushes to Convex `playlists` + `tracks`
3. **Remove fake TRACK_POOL** from `lib/vibefm.ts` once real tracks are seeded
4. **Cover art** — generate `.webp` per playlist, populate `coverArtFile`, `coverArtPrompt`, `coverArtDescription`

---

## vibefm-editorial.json

Each key is a playlist name matching `demo_playlists.name` in `catalog.db`. Fields:

| Field | Description |
|-------|-------------|
| `description` | 2–3 sentence card copy. The pitch. |
| `linerNotes` | First-person narrative essay. The story. See content spec. |
| `facts` | Array of exactly 3 facts. Educational, not atmospheric. |
| `curatorName` | The synthetic persona who "made" this playlist. |
| `curatorTitle` | Their role/title. Creative, playlist-specific. |

**Full content spec and writing guidelines:**
→ `nextjs-chat-template/docs/product/vibefm-playlist-content-spec.md`

The most important distinction: `description` is the pitch, `linerNotes` is a piece of writing that stands on its own. Read the spec before writing liner notes — the format is specific.

---

## How to add content

Open `vibefm-editorial.json` and fill in the fields for any playlist. Empty strings mean not yet written. The JSON keys must match the playlist names exactly as they appear in `catalog.db`.

When content is ready to ship, run the seed script in the vibefm repo to push to Convex.

---

## Playlist catalog

49 playlists across 8 mood categories. Full tracklists in `catalog.db` under `demo_playlists` and `demo_playlist_tracks`.

```sql
-- Get all tracks for a playlist
SELECT t.song_title, t.artist, t.youtube_id
FROM demo_playlist_tracks dpt
JOIN tracks t ON dpt.youtube_id = t.youtube_id
WHERE dpt.playlist_name = 'Hood Certified'
ORDER BY t.artist;
```

Query the tracklist before writing liner notes — not to reference songs by name, but to understand the emotional composition of the playlist. The themes and tensions in the track selection inform what the liner note is about.

---

## Liner note process

1. Query the tracklist for the playlist from `catalog.db`
2. Read the content spec: `nextjs-chat-template/docs/product/vibefm-playlist-content-spec.md`
3. Identify the emotional themes and tensions in the selection — without naming songs or artists
4. Write a first-person narrative set in the world the playlist name represents — not about music, about the human experience
5. End with a question that distills the piece
6. Add to `vibefm-editorial.json`
