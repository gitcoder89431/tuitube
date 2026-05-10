package commands

import "github.com/elpdev/tuipalette"

type Registry = tuipalette.Registry

func NewRegistry() *Registry {
	registry := tuipalette.NewRegistry()
	registry.SetDefaultModule(ModuleGlobal)
	return registry
}
