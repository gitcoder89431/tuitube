package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gitcoder89431/tuitube/internal/agentlog"
	"github.com/gitcoder89431/tuitube/internal/db"
	"github.com/gitcoder89431/tuitube/internal/player"
	tubesync "github.com/gitcoder89431/tuitube/internal/sync"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	database *db.DB
}

func New(database *db.DB) *Server {
	return &Server{database: database}
}

func (s *Server) Serve() error {
	srv := server.NewMCPServer(
		"tuitube",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	srv.AddTool(mcp.NewTool("search_tracks",
		mcp.WithDescription("Search the music library. Returns tracks matching query. Leave query empty to list recent tracks."),
		mcp.WithString("query", mcp.Description("Search query (artist, title, or both)")),
		mcp.WithBoolean("favorites_only", mcp.Description("Only return favorited tracks")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 30)")),
	), mcp.NewTypedToolHandler(s.searchTracks))

	srv.AddTool(mcp.NewTool("list_stations",
		mcp.WithDescription("List all synced YouTube channel stations"),
	), mcp.NewTypedToolHandler(s.listStations))

	srv.AddTool(mcp.NewTool("sync_station",
		mcp.WithDescription("Pull new tracks from YouTube for one or all stations via yt-dlp"),
		mcp.WithString("station_id", mcp.Description("Station ID to sync (omit to sync all)")),
	), mcp.NewTypedToolHandler(s.syncStation))

	srv.AddTool(mcp.NewTool("add_station",
		mcp.WithDescription("Add a new YouTube channel station and optionally sync it immediately"),
		mcp.WithString("url", mcp.Required(), mcp.Description("YouTube channel URL (any format: @handle, /channel/UC..., /c/, /user/)")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Display name for this station")),
		mcp.WithBoolean("sync_now", mcp.Description("Immediately sync all tracks after adding (default false)")),
	), mcp.NewTypedToolHandler(s.addStation))

	srv.AddTool(mcp.NewTool("play_track",
		mcp.WithDescription("Stream a track via mpv (no video, background). Stops any currently playing track first."),
		mcp.WithString("youtube_id", mcp.Required(), mcp.Description("YouTube video ID")),
		mcp.WithString("title", mcp.Description("Track title (for display only)")),
	), mcp.NewTypedToolHandler(s.playTrack))

	srv.AddTool(mcp.NewTool("stop_playback",
		mcp.WithDescription("Stop the currently playing track"),
	), mcp.NewTypedToolHandler(s.stopPlayback))

	srv.AddTool(mcp.NewTool("toggle_favorite",
		mcp.WithDescription("Toggle the favorite status of a track"),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track ID (from search_tracks result)")),
	), mcp.NewTypedToolHandler(s.toggleFavorite))

	srv.AddTool(mcp.NewTool("list_playlists",
		mcp.WithDescription("List all playlists (Favorites is always id=1)"),
	), mcp.NewTypedToolHandler(s.listPlaylists))

	srv.AddTool(mcp.NewTool("create_playlist",
		mcp.WithDescription("Create a new playlist"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Playlist name")),
	), mcp.NewTypedToolHandler(s.createPlaylist))

	srv.AddTool(mcp.NewTool("add_to_playlist",
		mcp.WithDescription("Add one or more tracks to a playlist in a single call"),
		mcp.WithNumber("playlist_id", mcp.Required(), mcp.Description("Playlist ID (from list_playlists)")),
		mcp.WithArray("track_ids", mcp.Required(), mcp.Description("Array of track IDs to add (from search_tracks)"), mcp.Items(map[string]any{"type": "string"})),
	), mcp.NewTypedToolHandler(s.addToPlaylist))

	srv.AddTool(mcp.NewTool("list_playlist_tracks",
		mcp.WithDescription("List tracks in a playlist"),
		mcp.WithNumber("playlist_id", mcp.Required(), mcp.Description("Playlist ID")),
	), mcp.NewTypedToolHandler(s.listPlaylistTracks))

	return server.ServeStdio(srv)
}

// --- tool args structs ---

type searchArgs struct {
	Query         string  `json:"query"`
	FavoritesOnly bool    `json:"favorites_only"`
	Limit         float64 `json:"limit"`
}

type syncArgs struct {
	StationID string `json:"station_id"`
}

type addStationArgs struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	SyncNow  bool   `json:"sync_now"`
}

type playArgs struct {
	YoutubeID string `json:"youtube_id"`
	Title     string `json:"title"`
}

type toggleFavArgs struct {
	TrackID string `json:"track_id"`
}

type createPlaylistArgs struct {
	Name string `json:"name"`
}

type playlistTrackArgs struct {
	PlaylistID float64  `json:"playlist_id"`
	TrackIDs   []string `json:"track_ids"`
}

type listPlaylistArgs struct {
	PlaylistID float64 `json:"playlist_id"`
}

// --- handlers ---

func (s *Server) searchTracks(_ context.Context, _ mcp.CallToolRequest, args searchArgs) (*mcp.CallToolResult, error) {
	limit := int(args.Limit)
	if limit <= 0 {
		limit = 30
	}
	tracks, err := s.database.ListTracks(args.Query, args.FavoritesOnly && args.Query == "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(tracks) > limit {
		tracks = tracks[:limit]
	}
	return jsonResult(tracks)
}

func (s *Server) listStations(_ context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
	stations, err := s.database.ListStations()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(stations)
}

func (s *Server) syncStation(_ context.Context, _ mcp.CallToolRequest, args syncArgs) (*mcp.CallToolResult, error) {
	stations, err := s.database.ListStations()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var results []map[string]any
	total := 0
	for _, st := range stations {
		if args.StationID != "" && st.ID != args.StationID {
			continue
		}
		var buf syncWriter
		n, err := tubesync.Station(s.database, st, &buf)
		result := map[string]any{"station": st.Name, "inserted": n, "log": buf.s}
		if err != nil {
			result["error"] = err.Error()
		}
		results = append(results, result)
		total += n
	}

	if total > 0 {
		if err := s.database.RebuildFTS(); err != nil {
			return mcp.NewToolResultError("fts rebuild: " + err.Error()), nil
		}
	}
	if total > 0 {
		agentlog.Write(fmt.Sprintf("↻ synced %d new tracks", total))
	}
	return jsonResult(map[string]any{"total_inserted": total, "stations": results})
}

func (s *Server) addStation(_ context.Context, _ mcp.CallToolRequest, args addStationArgs) (*mcp.CallToolResult, error) {
	var buf syncWriter
	station, err := tubesync.DiscoverStation(args.URL, &buf)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	station.Name = args.Name
	if err := s.database.UpsertStation(station); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]any{
		"station":  station,
		"log":      buf.s,
		"inserted": 0,
	}

	if args.SyncNow {
		var buf2 syncWriter
		n, err := tubesync.Station(s.database, station, &buf2)
		result["inserted"] = n
		result["sync_log"] = buf2.s
		if err != nil {
			result["sync_error"] = err.Error()
		} else if n > 0 {
			_ = s.database.RebuildFTS()
		}
	}

	return jsonResult(result)
}

func (s *Server) playTrack(_ context.Context, _ mcp.CallToolRequest, args playArgs) (*mcp.CallToolResult, error) {
	if err := player.Play(args.YoutubeID, args.Title, ""); err != nil {
		return mcp.NewToolResultError("mpv: " + err.Error()), nil
	}
	label := args.Title
	if label == "" {
		label = args.YoutubeID
	}
	agentlog.Write("▶ playing: " + label)
	return mcp.NewToolResultText(fmt.Sprintf("now playing: %s", label)), nil
}

func (s *Server) stopPlayback(_ context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
	np := player.NowPlaying()
	player.Stop()
	if np == nil {
		return mcp.NewToolResultText("nothing was playing"), nil
	}
	agentlog.Write("⏹ stopped: " + np.Title)
	return mcp.NewToolResultText("stopped: " + np.Title), nil
}

func (s *Server) toggleFavorite(_ context.Context, _ mcp.CallToolRequest, args toggleFavArgs) (*mcp.CallToolResult, error) {
	isFav, err := s.database.ToggleFavorite(args.TrackID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	state := "removed from favorites"
	if isFav {
		state = "added to favorites"
	}
	agentlog.Write("♥ " + state + ": " + args.TrackID)
	return mcp.NewToolResultText(state), nil
}

func (s *Server) listPlaylists(_ context.Context, _ mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
	playlists, err := s.database.ListPlaylists()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(playlists)
}

func (s *Server) createPlaylist(_ context.Context, _ mcp.CallToolRequest, args createPlaylistArgs) (*mcp.CallToolResult, error) {
	id, err := s.database.CreatePlaylist(args.Name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	agentlog.Write(fmt.Sprintf("♪ created playlist: %s (id=%d)", args.Name, id))
	return jsonResult(map[string]any{"id": id, "name": args.Name})
}

func (s *Server) addToPlaylist(_ context.Context, _ mcp.CallToolRequest, args playlistTrackArgs) (*mcp.CallToolResult, error) {
	pid := int64(args.PlaylistID)
	added := 0
	for _, id := range args.TrackIDs {
		if err := s.database.AddToPlaylist(pid, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed on %s: %v", id, err)), nil
		}
		added++
	}
	return mcp.NewToolResultText(fmt.Sprintf("added %d tracks", added)), nil
}

func (s *Server) listPlaylistTracks(_ context.Context, _ mcp.CallToolRequest, args listPlaylistArgs) (*mcp.CallToolResult, error) {
	tracks, err := s.database.ListPlaylistTracks(int64(args.PlaylistID))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(tracks)
}

// --- helpers ---

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// syncWriter captures sync log output into a string.
type syncWriter struct{ s string }

func (w *syncWriter) Write(p []byte) (int, error) {
	w.s += string(p)
	return len(p), nil
}
