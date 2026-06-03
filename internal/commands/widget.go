package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const paletteWidth = 60

// Options configures the palette widget.
type Options struct {
	Title              string
	Placeholder        string
	Modules            []string
	ReservedNamespaces []string
	Pages              map[string]Page
}

type innerPalette struct {
	registry   *Registry
	opts       Options
	styles     Styles
	ctx        Context
	query      string
	results    []Command
	selected   int
	activePage Page
	lastAction widgetAction
}

func newInnerPalette(r *Registry, opts Options) innerPalette {
	m := innerPalette{registry: r, opts: opts}
	m.refilter()
	return m
}

func (m innerPalette) Update(msg tea.Msg) (innerPalette, tea.Cmd) {
	m.lastAction = widgetAction{}
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if m.activePage != nil {
		page, action := m.activePage.Update(kp)
		m.activePage = page
		if action.Type != widgetActionNone {
			if action.Type == widgetActionBack {
				m.activePage = nil
			}
			m.lastAction = action
		}
		return m, nil
	}
	switch kp.String() {
	case "esc":
		m.lastAction = widgetAction{Type: widgetActionClose}
	case "enter":
		if len(m.results) > 0 {
			cmd := m.results[m.selected]
			if cmd.OpensPage != "" {
				if p, ok := m.opts.Pages[cmd.OpensPage]; ok {
					p.Reset()
					m.activePage = p
				}
			} else if cmd.Run != nil {
				m.lastAction = widgetAction{Type: widgetActionExecute, Command: &m.results[m.selected]}
			}
		}
	case "up", "ctrl+p":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "ctrl+n":
		if m.selected < len(m.results)-1 {
			m.selected++
		}
	case "backspace", "ctrl+h":
		if len(m.query) > 0 {
			runes := []rune(m.query)
			m.query = string(runes[:len(runes)-1])
			m.refilter()
		}
	default:
		if s := kp.String(); len([]rune(s)) == 1 {
			r := []rune(s)[0]
			if r >= 32 && r != 127 {
				m.query += s
				m.refilter()
			}
		}
	}
	return m, nil
}

func (m *innerPalette) refilter() {
	m.results = m.registry.Filter(m.query, m.ctx)
	if m.selected >= len(m.results) {
		if len(m.results) > 0 {
			m.selected = len(m.results) - 1
		} else {
			m.selected = 0
		}
	}
}

func (m innerPalette) Action() widgetAction { return m.lastAction }
func (m *innerPalette) SetStyles(s Styles)  { m.styles = s }
func (m *innerPalette) ClearAction()        { m.lastAction = widgetAction{} }

func (m *innerPalette) Reset(ctx Context) {
	m.ctx = ctx
	m.query = ""
	m.selected = 0
	m.activePage = nil
	m.lastAction = widgetAction{}
	m.refilter()
}

func (m innerPalette) View() string {
	if m.activePage != nil {
		return m.activePage.View(m.styles, paletteWidth)
	}
	return m.renderList()
}

func (m innerPalette) renderList() string {
	s := m.styles
	query := m.query
	if query == "" {
		query = s.Muted.Render(m.opts.Placeholder)
	}
	header := s.Title.Render(m.opts.Title) + "\n" + s.Accent.Render("> ") + query + "\n"

	if len(m.results) == 0 {
		return s.Modal.Width(paletteWidth).Render(header + "\n" + s.Muted.Render("  no commands match"))
	}

	const maxVisible = 10
	start := 0
	if m.selected >= maxVisible {
		start = m.selected - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.results) {
		end = len(m.results)
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for i := start; i < end; i++ {
		cmd := m.results[i]
		label := cmd.Title
		if cmd.Description != "" {
			label += "  " + s.Muted.Render(cmd.Description)
		}
		if i == m.selected {
			b.WriteString(s.Selected.Render("> " + label))
		} else {
			b.WriteString(s.Text.Render("  " + label))
		}
		b.WriteString("\n")
	}
	if end < len(m.results) {
		b.WriteString(s.Muted.Render("  ↓ more...") + "\n")
	}
	return s.Modal.Width(paletteWidth).Render(b.String())
}
