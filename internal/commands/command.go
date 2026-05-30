package commands

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	ModuleHome     = "home"
	ModuleSettings = "settings"
	ModuleHelp     = "help"
	ModuleLogs     = "logs"
	ModuleGlobal   = "global"
)

type Command struct {
	ID          string
	Module      string
	Title       string
	Description string
	Keywords    []string
	Order       int
	Run         func() tea.Cmd
	OpensPage   string
}

type Context struct {
	ActiveScreen string
}

type Styles struct {
	Modal    lipgloss.Style
	Title    lipgloss.Style
	Text     lipgloss.Style
	Muted    lipgloss.Style
	Selected lipgloss.Style
	Accent   lipgloss.Style
}

type SelectItem struct {
	Label   string
	Current bool
	Value   any
}

type widgetActionType int

const (
	widgetActionNone widgetActionType = iota
	widgetActionClose
	widgetActionExecute
	widgetActionBack
	widgetActionPage
)

type widgetAction struct {
	Type    widgetActionType
	Command *Command
	Page    string
	Data    any
}

// Page is a custom view that can be pushed onto the palette navigation stack.
type Page interface {
	Update(tea.KeyPressMsg) (Page, widgetAction)
	View(styles Styles, width int) string
	Reset()
}
