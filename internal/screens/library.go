package screens

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"

	"github.com/gitcoder89431/tui-tube/internal/db"
	"github.com/gitcoder89431/tui-tube/internal/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

// PlayTrackMsg asks the app to start streaming a track via mpv.
type PlayTrackMsg struct {
	YoutubeID string
	Title     string
	Artist    string
}

// DownloadTrackMsg asks the app to download a track via yt-dlp.
type DownloadTrackMsg struct{ YoutubeID, Title, Artist string }

// TogglePauseMsg asks the app to pause or resume current playback.
type TogglePauseMsg struct{}


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

	nowPlayingID     string
	nowPlayingPaused bool
}

func NewLibrary(database *db.DB, t theme.Theme) Library {
	return Library{database: database, theme: t}
}

func (l Library) WithTheme(t theme.Theme) Library {
	l.theme = t
	return l
}

func (l Library) Tracks() []db.Track { return l.tracks }
func (l Library) Cursor() int        { return l.cursor }

func (l Library) WithNowPlaying(youtubeID string, paused bool) Library {
	l.nowPlayingID = youtubeID
	l.nowPlayingPaused = paused
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
	if l.searchActive {
		return true
	}
	// claim esc when a search query is active so we can clear it
	return l.searchQuery != "" && msg.String() == "esc"
}

func (l Library) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if l.searchActive {
		return l.handleSearchKey(msg)
	}
	return l.handleTableKey(msg)
}

func (l Library) handleTableKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if l.searchQuery != "" {
			l.searchQuery = ""
			l.cursor = 0
			return l, l.loadCmd()
		}
	case "up", "k":
		l.cursor = max(0, l.cursor-1)
	case "down", "j":
		l.cursor = min(max(0, len(l.tracks)-1), l.cursor+1)
	case "enter":
		if t := l.selected(); t != nil {
			return l, func() tea.Msg {
				return PlayTrackMsg{YoutubeID: t.YoutubeID, Title: t.SongTitle, Artist: t.Artist}
			}
		}
	case "p":
		return l, func() tea.Msg { return TogglePauseMsg{} }
	case "d":
		if t := l.selected(); t != nil {
			return l, func() tea.Msg {
				return DownloadTrackMsg{YoutubeID: t.YoutubeID, Title: t.SongTitle, Artist: t.Artist}
			}
		}
	case " ", "space":
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
	case " ", "space":
		l.searchQuery += " "
		l.cursor = 0
		return l, l.loadCmd()
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
	return "  " +
		l.theme.Muted.Bold(true).Render(pad("Song", titleW)) +
		"  " +
		l.theme.Muted.Bold(true).Render(pad("Artist", artistW))
}

func (l Library) trackRow(t db.Track, selected bool, titleW, artistW, width int) string {
	playing := t.YoutubeID == l.nowPlayingID

	playSymbol := "  "
	if playing {
		if l.nowPlayingPaused {
			playSymbol = "⏸ "
		} else {
			playSymbol = "▶ "
		}
	}

	artistCell := pad(truncate(t.Artist, artistW), artistW)

	// cell renders text and artist segments with explicit theme colors so they're
	// consistent regardless of what ANSI state precedes them in the row.
	textFg := l.theme.Text.GetForeground()
	accentFg := l.theme.Accent.GetForeground()
	warnFg := l.theme.Warn.GetForeground()

	// favorited and playing rows use bright white so they stand out
	rowFg := textFg
	if t.IsFavorite || playing {
		rowFg = lipgloss.Color("#FFFFFF")
	}

	cell := func(s string, fg, bg color.Color) string {
		st := lipgloss.NewStyle().Foreground(fg)
		if bg != nil {
			st = st.Background(bg)
		}
		return st.Render(s)
	}

	if selected {
		selBg := l.theme.Selected.GetBackground()
		play := cell(playSymbol, rowFg, selBg)
		if playing {
			play = cell(playSymbol, warnFg, selBg)
		}
		var title string
		if t.IsFavorite {
			titleText := truncate(t.SongTitle, titleW-2)
			trailing := strings.Repeat(" ", max(0, titleW-lipgloss.Width(titleText)-2))
			title = cell(titleText, rowFg, selBg) + cell(" ♥", accentFg, selBg) + cell(trailing, rowFg, selBg)
		} else {
			title = cell(pad(truncate(t.SongTitle, titleW), titleW), rowFg, selBg)
		}
		return play + title + cell("  ", rowFg, selBg) + cell(artistCell, rowFg, selBg)
	}

	// unselected
	play := cell(playSymbol, rowFg, nil)
	if playing {
		play = cell(playSymbol, warnFg, nil)
	}
	var title string
	if t.IsFavorite {
		titleText := truncate(t.SongTitle, titleW-2)
		trailing := strings.Repeat(" ", max(0, titleW-lipgloss.Width(titleText)-2))
		title = cell(titleText, rowFg, nil) + cell(" ♥", accentFg, nil) + cell(trailing, rowFg, nil)
	} else {
		title = cell(pad(truncate(t.SongTitle, titleW), titleW), rowFg, nil)
	}
	return play + title + cell("  ", rowFg, nil) + cell(artistCell, rowFg, nil)
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

const scrolloff = 4 // rows to keep visible above/below cursor (Vim scrolloff)

func (l Library) computeOffset(visibleRows int) int {
	if visibleRows <= 0 || len(l.tracks) == 0 {
		return 0
	}
	total := len(l.tracks)
	soff := min(scrolloff, max(1, (visibleRows-1)/2))

	// near the top — pin to start, cursor naturally has fewer than soff rows above
	if l.cursor <= soff {
		return 0
	}
	// near the bottom — pin to end, cursor naturally has fewer than soff rows below
	if l.cursor >= total-soff {
		return max(0, total-visibleRows)
	}
	// in the middle — keep soff rows above the cursor
	return clamp(l.cursor-soff, 0, max(0, total-visibleRows))
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
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download")),
		key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "favorite")),
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "favorites")),
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
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// trim runes until display width fits, then append ellipsis
	runes := []rune(s)
	for i := len(runes) - 1; i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

func pad(s string, width int) string {
	sw := lipgloss.Width(s)
	if sw >= width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-sw)
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

