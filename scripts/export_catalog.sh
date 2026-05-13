#!/usr/bin/env bash
# Export the curated catalog (stations + tracks) from your working DB into a
# distributable catalog.db. Run this before cutting a release.
#
# Usage:
#   ./scripts/export_catalog.sh                      # uses defaults
#   ./scripts/export_catalog.sh --src ~/my.db --out ./catalog.db

set -euo pipefail

SRC="${SRC:-$HOME/.local/share/tuitube/tuitube.db}"
OUT="${OUT:-$(dirname "$0")/../catalog.db}"
VERSION="$(date -u +%Y%m%d)"

# parse flags
while [[ $# -gt 0 ]]; do
  case $1 in
    --src) SRC="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    *) echo "unknown flag: $1" >&2; exit 1 ;;
  esac
done

echo "source:  $SRC"
echo "output:  $OUT"
echo "version: $VERSION"
echo

rm -f "$OUT"

sqlite3 "$OUT" <<SQL
ATTACH DATABASE '$SRC' AS src;

CREATE TABLE stations (
  id                  TEXT PRIMARY KEY,
  name                TEXT NOT NULL,
  description         TEXT,
  color               TEXT,
  youtube_channel_id  TEXT NOT NULL,
  uploads_playlist_id TEXT NOT NULL,
  last_synced         INTEGER,
  created_at          INTEGER
);

CREATE TABLE tracks (
  id           TEXT PRIMARY KEY,
  station_id   TEXT NOT NULL REFERENCES stations(id),
  youtube_id   TEXT NOT NULL UNIQUE,
  song_title   TEXT,
  artist       TEXT,
  raw_title    TEXT NOT NULL,
  search_text  TEXT,
  thumbnail    TEXT,
  published_at INTEGER,
  created_at   INTEGER
);

CREATE INDEX tracks_station_idx    ON tracks(station_id);
CREATE INDEX tracks_published_idx  ON tracks(published_at DESC);
CREATE INDEX tracks_youtube_id_idx ON tracks(youtube_id);

CREATE VIRTUAL TABLE tracks_fts USING fts5(
  song_title, artist, raw_title,
  content='tracks', content_rowid='rowid'
);

CREATE TABLE meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

INSERT INTO stations SELECT * FROM src.stations;
INSERT INTO tracks   SELECT * FROM src.tracks;
INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild');
INSERT INTO meta VALUES ('catalog_version', '$VERSION');

-- demo playlists: shipped with catalog, seeded into new installs
CREATE TABLE demo_playlists (
  name TEXT PRIMARY KEY
);
CREATE TABLE demo_playlist_tracks (
  playlist_name TEXT NOT NULL REFERENCES demo_playlists(name) ON DELETE CASCADE,
  youtube_id    TEXT NOT NULL,
  PRIMARY KEY (playlist_name, youtube_id)
);

-- export all non-Favorites playlists as demos
INSERT INTO demo_playlists (name)
  SELECT name FROM src.playlists WHERE id != 1;

INSERT INTO demo_playlist_tracks (playlist_name, youtube_id)
  SELECT p.name, t.youtube_id
  FROM src.playlist_tracks pt
  JOIN src.playlists p ON p.id = pt.playlist_id
  JOIN src.tracks t ON t.id = pt.track_id
  WHERE p.id != 1;

DETACH DATABASE src;
SQL

TRACKS=$(sqlite3 "$OUT" "SELECT COUNT(*) FROM tracks;")
STATIONS=$(sqlite3 "$OUT" "SELECT COUNT(*) FROM stations;")

echo "exported:"
echo "  stations : $STATIONS"
echo "  tracks   : $TRACKS"
echo "  version  : $VERSION"
echo "  size     : $(du -h "$OUT" | cut -f1)"
echo
echo "attach to a GitHub release as catalog.db"
