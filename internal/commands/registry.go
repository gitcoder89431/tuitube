package commands

import "strings"

type Registry struct {
	defaultModule string
	commands      []Command
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) SetDefaultModule(m string) { r.defaultModule = m }

func (r *Registry) Register(cmd Command) {
	if cmd.Module == "" {
		cmd.Module = r.defaultModule
	}
	r.commands = append(r.commands, cmd)
}

func (r *Registry) Find(id string) (Command, bool) {
	for _, c := range r.commands {
		if c.ID == id {
			return c, true
		}
	}
	return Command{}, false
}

func (r *Registry) Filter(query string, _ Context) []Command {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []Command
	for _, c := range r.commands {
		if query == "" || cmdMatches(c, query) {
			out = append(out, c)
		}
	}
	return out
}

func cmdMatches(c Command, q string) bool {
	if strings.Contains(strings.ToLower(c.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Description), q) {
		return true
	}
	for _, kw := range c.Keywords {
		if strings.Contains(strings.ToLower(kw), q) {
			return true
		}
	}
	return false
}
