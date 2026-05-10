#!/usr/bin/env python3
"""
Import a Convex snapshot into the tui-tube SQLite database.

Usage:
    python3 scripts/import_convex.py \
        --snapshot ./temp/snapshot_*/  \
        --db ~/.local/share/tui-tube/tui-tube.db
"""

import argparse
import json
import os
import sqlite3
import sys
from pathlib import Path


SCHEMA = """
CREATE TABLE IF NOT EXISTS stations (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    description         TEXT,
    color               TEXT,
    youtube_channel_id  TEXT NOT NULL,
    uploads_playlist_id TEXT NOT NULL,
    last_synced         INTEGER,
    created_at          INTEGER
);

CREATE TABLE IF NOT EXISTS tracks (
    id          TEXT PRIMARY KEY,
    station_id  TEXT NOT NULL REFERENCES stations(id),
    youtube_id  TEXT NOT NULL UNIQUE,
    song_title  TEXT,
    artist      TEXT,
    raw_title   TEXT NOT NULL,
    search_text TEXT,
    thumbnail   TEXT,
    published_at INTEGER,
    created_at  INTEGER
);

CREATE INDEX IF NOT EXISTS tracks_station_idx    ON tracks(station_id);
CREATE INDEX IF NOT EXISTS tracks_published_idx  ON tracks(published_at DESC);
CREATE INDEX IF NOT EXISTS tracks_youtube_id_idx ON tracks(youtube_id);

CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(
    song_title,
    artist,
    raw_title,
    content='tracks',
    content_rowid='rowid'
);

CREATE TABLE IF NOT EXISTS playlists (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
    playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id    TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    added_at    INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
    PRIMARY KEY (playlist_id, track_id)
);
"""


def read_jsonl(path: Path):
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line:
                yield json.loads(line)


def import_stations(cur, snapshot: Path):
    path = snapshot / "stations" / "documents.jsonl"
    rows = list(read_jsonl(path))
    cur.executemany(
        """
        INSERT OR REPLACE INTO stations
            (id, name, description, color, youtube_channel_id, uploads_playlist_id, last_synced, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        """,
        [
            (
                r["_id"],
                r["name"],
                r.get("description"),
                r.get("color"),
                r["youtubeChannelId"],
                r["uploadsPlaylistId"],
                int(r["lastSynced"]) if r.get("lastSynced") else None,
                int(r["_creationTime"]),
            )
            for r in rows
        ],
    )
    print(f"  stations: {len(rows)} imported")
    return len(rows)


def import_tracks(cur, snapshot: Path):
    path = snapshot / "tracks" / "documents.jsonl"
    rows = list(read_jsonl(path))
    cur.executemany(
        """
        INSERT OR REPLACE INTO tracks
            (id, station_id, youtube_id, song_title, artist, raw_title, search_text, thumbnail, published_at, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        [
            (
                r["_id"],
                r["stationId"],
                r["youtubeId"],
                r.get("songTitle"),
                r.get("artist"),
                r["rawTitle"],
                r.get("searchText"),
                r.get("thumbnail"),
                int(r["publishedAt"]) if r.get("publishedAt") else None,
                int(r["_creationTime"]),
            )
            for r in rows
        ],
    )
    print(f"  tracks:   {len(rows)} imported")
    return len(rows)


def rebuild_fts(cur):
    cur.execute("INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild')")
    print("  fts5:     index rebuilt")


def seed_favorites(cur):
    cur.execute(
        "INSERT OR IGNORE INTO playlists (id, name) VALUES (1, 'Favorites')"
    )
    print("  playlists: Favorites seeded (id=1)")


def main():
    parser = argparse.ArgumentParser(description="Import Convex snapshot into tui-tube SQLite DB")
    parser.add_argument("--snapshot", required=True, help="Path to snapshot directory")
    parser.add_argument("--db", required=True, help="Path to output SQLite database")
    args = parser.parse_args()

    snapshot = Path(args.snapshot).expanduser().resolve()
    db_path = Path(args.db).expanduser().resolve()

    if not snapshot.is_dir():
        print(f"error: snapshot directory not found: {snapshot}", file=sys.stderr)
        sys.exit(1)

    if not (snapshot / "tracks" / "documents.jsonl").exists():
        print(f"error: tracks/documents.jsonl not found in snapshot", file=sys.stderr)
        sys.exit(1)

    db_path.parent.mkdir(parents=True, exist_ok=True)

    print(f"snapshot: {snapshot}")
    print(f"database: {db_path}")
    print()

    con = sqlite3.connect(db_path)
    con.execute("PRAGMA journal_mode=WAL")
    con.execute("PRAGMA foreign_keys=ON")

    try:
        cur = con.cursor()
        cur.executescript(SCHEMA)

        print("importing...")
        import_stations(cur, snapshot)
        import_tracks(cur, snapshot)
        rebuild_fts(cur)
        seed_favorites(cur)
        con.commit()

        # summary
        print()
        print("done.")
        for table in ("stations", "tracks", "playlists", "playlist_tracks"):
            (count,) = cur.execute(f"SELECT COUNT(*) FROM {table}").fetchone()
            print(f"  {table:<20} {count:>6} rows")

    except Exception as e:
        con.rollback()
        print(f"\nerror: {e}", file=sys.stderr)
        sys.exit(1)
    finally:
        con.close()


if __name__ == "__main__":
    main()
