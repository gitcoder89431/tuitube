package commands

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/gitcoder89431/tuitube/internal/theme"
)

const pageThemes = "themes"

type PaletteModel struct {
	registry *Registry
	themes   []theme.Theme
	inner    innerPalette
	ctx      Context
	original string
	executed *Command
	action   PaletteAction
}

type PaletteAction struct {
	Type    PaletteActionType
	Command *Command
	Theme   *theme.Theme
}

type PaletteActionType int

const (
	PaletteActionNone PaletteActionType = iota
	PaletteActionClose
	PaletteActionExecute
	PaletteActionPreviewTheme
	PaletteActionConfirmTheme
	PaletteActionCancelTheme
)

func NewPaletteModel(registry *Registry, themes []theme.Theme) PaletteModel {
	m := PaletteModel{registry: registry, themes: themes}
	m.rebuildInner()
	return m
}

func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	m.executed = nil
	m.action = PaletteAction{}
	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	m.translateAction(m.inner.Action())
	return m, cmd
}

func (m PaletteModel) View(t theme.Theme) string {
	m.inner.SetStyles(stylesFromTheme(t))
	return m.inner.View()
}

func (m *PaletteModel) Reset(currentTheme string, ctx Context) {
	m.ctx = ctx
	m.original = currentTheme
	m.executed = nil
	m.action = PaletteAction{}
	m.rebuildInner()
	m.inner.Reset(ctx)
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }
func (m PaletteModel) Action() PaletteAction     { return m.action }

func (m *PaletteModel) ClearAction() {
	m.action = PaletteAction{}
	m.inner.ClearAction()
}

func (m *PaletteModel) rebuildInner() {
	m.inner = newInnerPalette(m.registry, Options{
		Title:              "tuitube",
		Placeholder:        "type a command...",
		Modules:            []string{ModuleHome, ModuleSettings, ModuleHelp, ModuleLogs, ModuleGlobal},
		ReservedNamespaces: []string{ModuleHome, ModuleSettings, ModuleHelp, ModuleLogs},
		Pages: map[string]Page{
			pageThemes: newThemePage(m.themes, m.original),
		},
	})
}

func (m *PaletteModel) translateAction(action widgetAction) {
	switch action.Type {
	case widgetActionClose:
		m.action = PaletteAction{Type: PaletteActionClose}
	case widgetActionExecute:
		if action.Command == nil {
			return
		}
		command, ok := m.registry.Find(action.Command.ID)
		if !ok {
			return
		}
		m.executed = &command
		m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
	case widgetActionBack:
		if action.Page == "theme-cancel" {
			if selected, ok := action.Data.(theme.Theme); ok {
				m.action = PaletteAction{Type: PaletteActionCancelTheme, Theme: &selected}
			}
		}
	case widgetActionPage:
		selected, ok := action.Data.(theme.Theme)
		if !ok {
			return
		}
		switch action.Page {
		case "theme-preview":
			m.action = PaletteAction{Type: PaletteActionPreviewTheme, Theme: &selected}
		case "theme-confirm":
			m.action = PaletteAction{Type: PaletteActionConfirmTheme, Theme: &selected}
		}
	}
}

func stylesFromTheme(t theme.Theme) Styles {
	return Styles{
		Modal:    t.Modal,
		Title:    t.Title,
		Text:     t.Text,
		Muted:    t.Muted,
		Selected: t.Selected,
		Accent:   t.PaletteAccent,
	}
}

func newThemePage(themes []theme.Theme, original string) *windowedThemePage {
	items := make([]SelectItem, 0, len(themes))
	selected := 0
	var cancelData any
	for i, candidate := range themes {
		current := candidate.Name == original
		if current {
			selected = i
			cancelData = candidate
		}
		items = append(items, SelectItem{Label: candidate.Name, Current: current, Value: candidate})
	}
	p := &windowedThemePage{
		title:      "tuitube / Themes",
		subtitle:   "↑↓ preview · enter select · esc back",
		items:      items,
		selected:   selected,
		initial:    selected,
		cancelData: cancelData,
		maxVisible: 12,
	}
	p.clampOffset()
	return p
}

type windowedThemePage struct {
	title      string
	subtitle   string
	items      []SelectItem
	selected   int
	initial    int
	offset     int
	maxVisible int
	cancelData any
}

func (p *windowedThemePage) clampOffset() {
	if p.selected < p.offset {
		p.offset = p.selected
	}
	if p.selected >= p.offset+p.maxVisible {
		p.offset = p.selected - p.maxVisible + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p *windowedThemePage) Update(msg tea.KeyPressMsg) (Page, widgetAction) {
	switch msg.String() {
	case "esc", "backspace", "ctrl+h":
		return p, widgetAction{Type: widgetActionBack, Page: "theme-cancel", Data: p.cancelData}
	case "up", "ctrl+p":
		if p.selected > 0 {
			p.selected--
			p.clampOffset()
			return p, p.preview()
		}
	case "down", "ctrl+n":
		if p.selected < len(p.items)-1 {
			p.selected++
			p.clampOffset()
			return p, p.preview()
		}
	case "enter":
		if len(p.items) > 0 {
			return p, widgetAction{Type: widgetActionPage, Page: "theme-confirm", Data: p.items[p.selected].Value}
		}
	}
	return p, widgetAction{}
}

func (p *windowedThemePage) preview() widgetAction {
	if len(p.items) == 0 {
		return widgetAction{}
	}
	return widgetAction{Type: widgetActionPage, Page: "theme-preview", Data: p.items[p.selected].Value}
}

func (p *windowedThemePage) View(styles Styles, width int) string {
	innerWidth := width - styles.Modal.GetHorizontalFrameSize()

	rendered := styles.Title.Render(p.title)
	slashCount := max(3, innerWidth-lipgloss.Width(rendered)-1)
	header := rendered + " " + styles.Accent.Render(strings.Repeat("/", slashCount))
	if p.subtitle != "" {
		header += "\n" + styles.Muted.Width(innerWidth).Render(p.subtitle)
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	end := p.offset + p.maxVisible
	if end > len(p.items) {
		end = len(p.items)
	}
	for i := p.offset; i < end; i++ {
		item := p.items[i]
		label := item.Label
		if item.Current {
			label += "  •"
		}
		if i == p.selected {
			b.WriteString(styles.Selected.Render("> " + label))
		} else {
			b.WriteString(styles.Text.Render("  " + label))
		}
		b.WriteString("\n")
	}
	if end < len(p.items) {
		b.WriteString(styles.Muted.Render(fmt.Sprintf("  ↓ %d more", len(p.items)-end)) + "\n")
	}
	return styles.Modal.Width(width).Render(b.String())
}

func (p *windowedThemePage) Reset() {
	p.selected = p.initial
	p.clampOffset()
}
