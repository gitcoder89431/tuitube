package screens

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gitcoder89431/tui-tube/internal/db"
	"github.com/gitcoder89431/tui-tube/internal/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

// PlaylistSelectedMsg asks the app to load a playlist into the library.
type PlaylistSelectedMsg struct {
	Title  string
	Loader func() ([]db.Track, error)
}

// PlaylistsLoadedMsg carries playlists + stations for display.
type PlaylistsLoadedMsg struct {
	Playlists []db.Playlist
	Stations  []db.StationSummary
	Err       error
}

type playlistEntry struct {
	label      string
	count      int
	isHeader   bool
	loader     func() ([]db.Track, error)
}

type Playlists struct {
	database *db.DB
	theme    theme.Theme

	entries     []playlistEntry
	cursor      int
	creating    bool   // name-entry mode for new playlist
	newName     string
	err         error
}

func NewPlaylists(database *db.DB, t theme.Theme) Playlists {
	return Playlists{database: database, theme: t}
}

func (s Playlists) WithTheme(t theme.Theme) Playlists {
	s.theme = t
	return s
}

func (s Playlists) Init() tea.Cmd {
	return s.loadCmd()
}

func (s Playlists) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case PlaylistsLoadedMsg:
		if msg.Err != nil {
			s.err = msg.Err
			return s, nil
		}
		s.entries = buildEntries(msg.Playlists, msg.Stations, s.database)
		s.cursor = clampToSelectable(s.entries, 0)
		return s, nil

	case tea.KeyPressMsg:
		if s.creating {
			return s.handleNameKey(msg)
		}
		return s.handleKey(msg)
	}
	return s, nil
}

func (s Playlists) CapturesKey(msg tea.KeyPressMsg) bool {
	// claim n so global "next song" doesn't fire when on the playlists screen
	return s.creating || msg.String() == "n"
}

func (s Playlists) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		s.cursor = prevSelectable(s.entries, s.cursor)
	case "down", "j":
		s.cursor = nextSelectable(s.entries, s.cursor)
	case "enter":
		if e := s.selected(); e != nil {
			loader := e.loader
			title := e.label
			return s, func() tea.Msg {
				return PlaylistSelectedMsg{Title: title, Loader: loader}
			}
		}
	case "n":
		s.creating = true
		s.newName = ""
	}
	return s, nil
}

func (s Playlists) handleNameKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.creating = false
		s.newName = ""
	case "enter":
		if name := strings.TrimSpace(s.newName); name != "" {
			s.creating = false
			database := s.database
			return s, func() tea.Msg {
				if _, err := database.CreatePlaylist(name); err != nil {
					return PlaylistsLoadedMsg{Err: err}
				}
				playlists, err := database.ListPlaylists()
				stations, err2 := database.ListStationSummaries()
				if err != nil {
					return PlaylistsLoadedMsg{Err: err}
				}
				if err2 != nil {
					return PlaylistsLoadedMsg{Err: err2}
				}
				return PlaylistsLoadedMsg{Playlists: playlists, Stations: stations}
			}
		}
	case "backspace", "ctrl+h":
		if len(s.newName) > 0 {
			runes := []rune(s.newName)
			s.newName = string(runes[:len(runes)-1])
		}
	case " ", "space":
		s.newName += " "
	default:
		for _, r := range msg.String() {
			if unicode.IsPrint(r) {
				s.newName += string(r)
			}
		}
	}
	return s, nil
}

func (s Playlists) View(width, height int) string {
	if s.err != nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render(fmt.Sprintf("error: %v", s.err))
	}
	if s.entries == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("loading...")
	}

	promptLines := 0
	if s.creating {
		promptLines = 2
	}
	listLines := max(1, height-promptLines)

	selBg := s.theme.Selected.GetBackground()
	textFg := s.theme.Text.GetForeground()

	var lines []string
	for i, e := range s.entries {
		if e.isHeader {
			labelW := lipgloss.Width(e.label)
			fill := strings.Repeat("/", max(0, width-labelW-1))
			line := s.theme.Title.Render(e.label) + s.theme.PaletteAccent.Render(" "+fill)
			lines = append(lines, line)
			continue
		}
		countStr := fmt.Sprintf("%d tracks", e.count)
		gap := max(0, width-lipgloss.Width(e.label)-lipgloss.Width(countStr)-2)
		row := "  " + e.label + strings.Repeat(" ", gap-1) + s.theme.Muted.Render(countStr)

		if i == s.cursor {
			row = lipgloss.NewStyle().Foreground(textFg).Background(selBg).Width(width).Render(
				"  " + e.label + strings.Repeat(" ", gap-1) + countStr,
			)
		}
		lines = append(lines, row)
	}

	// pad to fill
	for len(lines) < listLines {
		lines = append(lines, "")
	}
	body := strings.Join(lines[:min(listLines, len(lines))], "\n")

	if !s.creating {
		return lipgloss.NewStyle().Width(width).Height(height).Render(body)
	}

	prompt := s.theme.Border.Render(strings.Repeat("─", width)) + "\n"
	cursor := s.theme.Accent.Render("█")
	prompt += s.theme.Accent.Render("New playlist: ") + s.newName + cursor
	return body + "\n" + prompt
}

func (s Playlists) Title() string { return "Playlists" }

func (s Playlists) KeyBindings() []key.Binding { return nil }

func (s Playlists) loadCmd() tea.Cmd {
	database := s.database
	return func() tea.Msg {
		playlists, err := database.ListPlaylists()
		if err != nil {
			return PlaylistsLoadedMsg{Err: err}
		}
		stations, err := database.ListStationSummaries()
		return PlaylistsLoadedMsg{Playlists: playlists, Stations: stations, Err: err}
	}
}

func (s Playlists) selected() *playlistEntry {
	if s.cursor < 0 || s.cursor >= len(s.entries) {
		return nil
	}
	e := s.entries[s.cursor]
	if e.isHeader {
		return nil
	}
	return &e
}

func buildEntries(playlists []db.Playlist, stations []db.StationSummary, database *db.DB) []playlistEntry {
	var entries []playlistEntry

	// favorites first (playlist id=1), then other user playlists
	entries = append(entries, playlistEntry{label: "Your Playlists", isHeader: true})

	for _, p := range playlists {
		if p.ID == 1 {
			entries = append(entries, playlistEntry{
				label: p.Name + " ♥",
				count: p.TrackCount,
				loader: func() ([]db.Track, error) {
					return database.ListPlaylistTracks(1)
				},
			})
		}
	}

	var userPlaylists []db.Playlist
	for _, p := range playlists {
		if p.ID != 1 {
			userPlaylists = append(userPlaylists, p)
		}
	}

	for _, p := range userPlaylists {
		pid := p.ID
		pname := p.Name
		entries = append(entries, playlistEntry{
			label: pname,
			count: p.TrackCount,
			loader: func() ([]db.Track, error) {
				return database.ListPlaylistTracks(pid)
			},
		})
	}

	if len(stations) > 0 {
		entries = append(entries, playlistEntry{label: "Stations", isHeader: true})
		for _, st := range stations {
			sid := st.ID
			sname := st.Name
			entries = append(entries, playlistEntry{
				label: sname,
				count: st.TrackCount,
				loader: func() ([]db.Track, error) {
					return database.ListStationTracks(sid)
				},
			})
		}
	}

	return entries
}

func clampToSelectable(entries []playlistEntry, start int) int {
	for i := start; i < len(entries); i++ {
		if !entries[i].isHeader {
			return i
		}
	}
	return start
}

func nextSelectable(entries []playlistEntry, cur int) int {
	for i := cur + 1; i < len(entries); i++ {
		if !entries[i].isHeader {
			return i
		}
	}
	return cur
}

func prevSelectable(entries []playlistEntry, cur int) int {
	for i := cur - 1; i >= 0; i-- {
		if !entries[i].isHeader {
			return i
		}
	}
	return cur
}
