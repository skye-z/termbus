package command

import (
	"strings"
)

// ParsedCommand describes a parsed command line.
type ParsedCommand struct {
	Name  string
	Args  []string
	Flags map[string]string
}

// Parser handles command parsing and alias expansion.
type Parser struct {
	aliases map[string]string
}

// NewParser creates a parser.
func NewParser() *Parser {
	return &Parser{aliases: make(map[string]string)}
}

// AddAlias registers an alias expansion.
func (p *Parser) AddAlias(alias, command string) {
	if alias == "" || command == "" {
		return
	}
	p.aliases[alias] = command
}

// Parse parses a command line into name, args, and flags.
func (p *Parser) Parse(input string) *ParsedCommand {
	line := strings.TrimSpace(input)
	if line == "" {
		return nil
	}

	if strings.HasPrefix(line, ":") {
		line = strings.TrimPrefix(line, ":")
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	if expansion, ok := p.aliases[parts[0]]; ok {
		parts = append(strings.Fields(expansion), parts[1:]...)
	}

	cmd := &ParsedCommand{Name: parts[0], Flags: make(map[string]string)}
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.HasPrefix(part, "--") {
			kv := strings.SplitN(strings.TrimPrefix(part, "--"), "=", 2)
			if len(kv) == 2 {
				cmd.Flags[kv[0]] = kv[1]
			} else {
				cmd.Flags[kv[0]] = "true"
			}
			continue
		}
		if strings.HasPrefix(part, "-") && len(part) > 1 {
			flag := strings.TrimPrefix(part, "-")
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
				cmd.Flags[flag] = parts[i+1]
				i++
			} else {
				cmd.Flags[flag] = "true"
			}
			continue
		}
		cmd.Args = append(cmd.Args, part)
	}

	return cmd
}
