package commandbar

import (
	"path/filepath"
	"strings"
)

// Command describes a command and its arguments.
type Command struct {
	Name        string
	Aliases     []string
	Description string
	Args        []string
}

// Completer provides command completion and history.
type Completer struct {
	commands map[string]*Command
	history  []string
	hosts    []string
	last     []string
}

// NewCompleter creates a new completer.
func NewCompleter() *Completer {
	return &Completer{commands: make(map[string]*Command)}
}

// AddCommand registers a command for completion.
func (c *Completer) AddCommand(cmd *Command) {
	if cmd == nil || cmd.Name == "" {
		return
	}
	c.commands[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		c.commands[alias] = cmd
	}
}

// AddHistory records a command in history.
func (c *Completer) AddHistory(cmd string) {
	if cmd == "" {
		return
	}
	c.history = append(c.history, cmd)
}

// SetHosts sets hostnames for completion.
func (c *Completer) SetHosts(hosts []string) {
	c.hosts = hosts
}

// Complete returns a list of completions for input.
func (c *Completer) Complete(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		c.last = nil
		return nil
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil
	}

	if len(parts) == 1 && strings.HasPrefix(parts[0], ":") {
		prefix := strings.TrimPrefix(parts[0], ":")
		c.last = c.matchCommand(prefix, ":")
		return c.last
	}

	if len(parts) == 1 {
		c.last = c.matchCommand(parts[0], "")
		return c.last
	}

	arg := parts[len(parts)-1]
	if strings.HasPrefix(arg, "~") || strings.HasPrefix(arg, "/") || strings.Contains(arg, string(filepath.Separator)) {
		c.last = c.matchPath(arg)
		return c.last
	}

	if strings.HasPrefix(arg, "@"); len(c.hosts) > 0 {
		prefix := strings.TrimPrefix(arg, "@")
		c.last = c.matchHosts(prefix)
		return c.last
	}

	c.last = nil
	return nil
}

// Last returns the last completion list.
func (c *Completer) Last() []string {
	return c.last
}

func (c *Completer) matchCommand(prefix, lead string) []string {
	results := make([]string, 0)
	for name := range c.commands {
		if strings.HasPrefix(name, prefix) {
			results = append(results, lead+name)
		}
	}
	return results
}

func (c *Completer) matchHosts(prefix string) []string {
	results := make([]string, 0)
	for _, host := range c.hosts {
		if strings.HasPrefix(host, prefix) {
			results = append(results, "@"+host)
		}
	}
	return results
}

func (c *Completer) matchPath(prefix string) []string {
	if strings.HasPrefix(prefix, "~") {
		prefix = strings.TrimPrefix(prefix, "~")
	}
	base := filepath.Dir(prefix)
	pattern := prefix + "*"
	if base == "." {
		pattern = prefix + "*"
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}
