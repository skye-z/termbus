package command

import (
	"fmt"
	"strings"
)

// CommandParser 命令解析器
type CommandParser struct {
	registry *CommandRegistry
}

// NewCommandParser 创建命令解析器
func NewCommandParser(registry *CommandRegistry) *CommandParser {
	return &CommandParser{
		registry: registry,
	}
}

// ParsedCommand 解析后的命令
type ParsedCommand struct {
	Command string
	Args    []string
	Flags   map[string]string
	Raw     string
}

// Parse 解析命令
func (p *CommandParser) Parse(input string) (*ParsedCommand, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty command")
	}

	var cmdStr string
	var args []string
	flags := make(map[string]string)

	tokens := p.tokenize(input)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no command")
	}

	cmdStr = tokens[0]

	for i := 1; i < len(tokens); i++ {
		token := tokens[i]
		if strings.HasPrefix(token, "--") {
			parts := strings.SplitN(token[2:], "=", 2)
			if len(parts) == 2 {
				flags[parts[0]] = parts[1]
			} else {
				flags[parts[0]] = "true"
			}
		} else if strings.HasPrefix(token, "-") && len(token) == 2 {
			flags[token[1:]] = "true"
		} else {
			args = append(args, token)
		}
	}

	return &ParsedCommand{
		Command: cmdStr,
		Args:    args,
		Flags:   flags,
		Raw:     input,
	}, nil
}

// Validate 验证命令
func (p *CommandParser) Validate(cmd *ParsedCommand) error {
	_, err := p.registry.Get(cmd.Command)
	return err
}

// tokenize 分词
func (p *CommandParser) tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false

	for _, r := range input {
		switch r {
		case '"':
			inQuote = !inQuote
		case ' ':
			if !inQuote && current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// BuildCommand 构建命令字符串
func BuildCommand(cmd string, args []string, flags map[string]string) string {
	var parts []string
	parts = append(parts, cmd)

	for key, value := range flags {
		if value == "true" {
			parts = append(parts, fmt.Sprintf("--%s", key))
		} else {
			parts = append(parts, fmt.Sprintf("--%s=%s", key, value))
		}
	}

	parts = append(parts, args...)

	return strings.Join(parts, " ")
}
