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

// PlayTrackMsg asks the app to start streaming a track via mpv.
type PlayTrackMsg struct{ YoutubeID string }

// DownloadTrackMsg asks the app to download a track via yt-dlp.
type DownloadTrackMsg struct{ YoutubeID, Title, Artist string }

// TracksLoadedMsg carries the result of a DB track query.
type TracksLoadedMsg struct {
	Tracks []db.Track
	Err    error
}

// FavToggledMsg carries the result of toggling a favorite.
type FavToggledMsg struct {
	TrackID string
	IsFav   bool
	Err     error
}

type Library struct {
	database *db.DB
	theme    theme.Theme

	tracks        []db.Track
	cursor        int
	searchQuery   string
	searchActive  bool
	favoritesOnly bool
	err           error
}

func NewLibrary(database *db.DB, t theme.Theme) Library {
	return Library{database: database, theme: t}
}

func (l Library) WithTheme(t theme.Theme) Library {
	l.theme = t
	return l
}

func (l Library) Init() tea.Cmd {
	return l.loadCmd()
}

func (l Library) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case TracksLoadedMsg:
		if msg.Err != nil {
			l.err = msg.Err
			return l, nil
		}
		l.tracks = msg.Tracks
		l.cursor = clamp(l.cursor, 0, max(0, len(l.tracks)-1))
		return l, nil

	case FavToggledMsg:
		if msg.Err != nil {
			return l, nil
		}
		for i, t := range l.tracks {
			if t.ID == msg.TrackID {
				l.tracks[i].IsFavorite = msg.IsFav
				// if in favorites-only view and unfavorited, remove from list
				if l.favoritesOnly && !msg.IsFav {
					l.tracks = append(l.tracks[:i], l.tracks[i+1:]...)
					l.cursor = clamp(l.cursor, 0, max(0, len(l.tracks)-1))
				}
				break
			}
		}
		return l, nil

	case tea.KeyPressMsg:
		return l.handleKey(msg)
	}
	return l, nil
}

// CapturesKey implements tuimod.KeyCapturer.
// When search is active, all keys go to the library screen before global handlers.
func (l Library) CapturesKey(msg tea.KeyPressMsg) bool {
	return l.searchActive
}

func (l Library) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if l.searchActive {
		return l.handleSearchKey(msg)
	}
	return l.handleTableKey(msg)
}

func (l Library) handleTableKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		l.cursor = max(0, l.cursor-1)
	case "down", "j":
		l.cursor = min(max(0, len(l.tracks)-1), l.cursor+1)
	case "enter":
		if t := l.selected(); t != nil {
			return l, func() tea.Msg { return PlayTrackMsg{YoutubeID: t.YoutubeID} }
		}
	case "d":
		if t := l.selected(); t != nil {
			return l, func() tea.Msg {
				return DownloadTrackMsg{YoutubeID: t.YoutubeID, Title: t.SongTitle, Artist: t.Artist}
			}
		}
	case " ":
		if t := l.selected(); t != nil {
			return l, toggleFavCmd(l.database, t.ID)
		}
	case "f":
		l.favoritesOnly = !l.favoritesOnly
		l.cursor = 0
		return l, l.loadCmd()
	case "/":
		l.searchActive = true
		l.searchQuery = ""
	}
	return l, nil
}

func (l Library) handleSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		l.searchActive = false
		if l.searchQuery != "" {
			l.searchQuery = ""
			l.cursor = 0
			return l, l.loadCmd()
		}
	case "enter":
		l.searchActive = false
	case "backspace", "ctrl+h":
		if len(l.searchQuery) > 0 {
			runes := []rune(l.searchQuery)
			l.searchQuery = string(runes[:len(runes)-1])
			l.cursor = 0
			return l, l.loadCmd()
		}
	default:
		// append printable characters to search query
		for _, r := range msg.String() {
			if unicode.IsPrint(r) {
				l.searchQuery += string(r)
				l.cursor = 0
				return l, l.loadCmd()
			}
		}
	}
	return l, nil
}

func (l Library) View(width, height int) string {
	if l.err != nil {
		return lipgloss.NewStyle().Width(width).Height(height).
			Render(fmt.Sprintf("error: %v\n\nIs the database path correct?", l.err))
	}
	if l.tracks == nil {
		return lipgloss.NewStyle().Width(width).Height(height).Render("loading...")
	}

	favW := 2
	artistW := 28
	if width < 70 {
		artistW = 18
	}
	titleW := max(8, width-favW-2-artistW-2)

	searchBarLines := 0
	if l.searchActive {
		searchBarLines = 2
	}
	tableLines := max(1, height-searchBarLines)
	dataRows := max(0, tableLines-1) // minus header row

	offset := l.computeOffset(dataRows)

	var lines []string

	// header
	lines = append(lines, l.headerRow(titleW, artistW, width))

	// track rows
	end := min(offset+dataRows, len(l.tracks))
	for i := offset; i < end; i++ {
		lines = append(lines, l.trackRow(l.tracks[i], i == l.cursor, titleW, artistW, width))
	}

	// blank padding rows
	for len(lines) < tableLines {
		lines = append(lines, strings.Repeat(" ", width))
	}

	// search bar
	if l.searchActive {
		lines = append(lines, l.theme.Border.Render(strings.Repeat("─", width)))
		lines = append(lines, l.searchBar(width))
	}

	return strings.Join(lines, "\n")
}

func (l Library) headerRow(titleW, artistW, width int) string {
	fav := l.theme.Muted.Render("♥ ")
	title := l.theme.Muted.Bold(true).Render(pad("Song", titleW))
	sep := l.theme.Muted.Render("  ")
	artist := l.theme.Muted.Bold(true).Render(pad("Artist", artistW))
	return fav + title + sep + artist
}

func (l Library) trackRow(t db.Track, selected bool, titleW, artistW, width int) string {
	fav := "  "
	if t.IsFavorite {
		fav = l.theme.Accent.Render("♥ ")
	}
	title := truncate(t.SongTitle, titleW)
	artist := truncate(t.Artist, artistW)

	row := fav + pad(title, titleW) + "  " + pad(artist, artistW)

	if selected {
		row = l.theme.Selected.Width(width).Render(row)
	}
	return row
}

func (l Library) searchBar(width int) string {
	prefix := l.theme.Accent.Render("/ ")
	query := l.searchQuery
	cursor := l.theme.Accent.Render("█")
	content := prefix + query + cursor
	available := width - lipgloss.Width(prefix) - lipgloss.Width(cursor)
	if lipgloss.Width(query) > available {
		runes := []rune(query)
		query = string(runes[len(runes)-available:])
		content = prefix + query + cursor
	}
	return content
}

func (l Library) computeOffset(visibleRows int) int {
	if visibleRows <= 0 || len(l.tracks) == 0 {
		return 0
	}
	offset := 0
	if l.cursor < offset {
		offset = l.cursor
	}
	if l.cursor >= visibleRows {
		offset = l.cursor - visibleRows + 1
	}
	return clamp(offset, 0, max(0, len(l.tracks)-visibleRows))
}

func (l Library) selected() *db.Track {
	if len(l.tracks) == 0 || l.cursor >= len(l.tracks) {
		return nil
	}
	t := l.tracks[l.cursor]
	return &t
}

func (l Library) Title() string {
	switch {
	case l.searchQuery != "":
		return fmt.Sprintf("Library · %q", l.searchQuery)
	case l.favoritesOnly:
		return "Library · Favorites"
	default:
		count := ""
		if len(l.tracks) > 0 {
			count = fmt.Sprintf(" (%d)", len(l.tracks))
		}
		return "Library" + count
	}
}

func (l Library) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "favorite")),
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "toggle favorites")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	}
}

// loadCmd returns a tea.Cmd that queries the DB with current filters.
func (l Library) loadCmd() tea.Cmd {
	query := l.searchQuery
	favOnly := l.favoritesOnly && query == ""
	database := l.database
	return func() tea.Msg {
		tracks, err := database.ListTracks(query, favOnly)
		return TracksLoadedMsg{Tracks: tracks, Err: err}
	}
}

func toggleFavCmd(database *db.DB, trackID string) tea.Cmd {
	return func() tea.Msg {
		isFav, err := database.ToggleFavorite(trackID)
		return FavToggledMsg{TrackID: trackID, IsFav: isFav, Err: err}
	}
}

func truncate(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	return string(runes[:maxWidth-1]) + "…"
}

func pad(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

