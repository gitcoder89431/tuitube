package commands

import "github.com/elpdev/tuipalette"

const (
	ModuleHome     = "home"
	ModuleSettings = "settings"
	ModuleHelp     = "help"
	ModuleLogs     = "logs"
	ModuleGlobal   = "global"
)

type Command = tuipalette.Command
type Context = tuipalette.Context
