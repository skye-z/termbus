package cli

import (
	"flag"
	"fmt"
)

// CLI provides basic plugin tooling.
type CLI struct {
	pluginPath string
	action     string
}

// NewCLI creates a CLI instance.
func NewCLI() *CLI {
	return &CLI{}
}

// Parse parses CLI arguments.
func (c *CLI) Parse(args []string) error {
	fs := flag.NewFlagSet("plugin", flag.ContinueOnError)
	fs.StringVar(&c.pluginPath, "plugin", "", "Plugin path")
	fs.StringVar(&c.action, "action", "", "Action: validate/build/sign")
	return fs.Parse(args)
}

// Run executes CLI action.
func (c *CLI) Run() error {
	switch c.action {
	case "validate":
		return nil
	case "build":
		return nil
	case "sign":
		return nil
	default:
		return fmt.Errorf("unknown action: %s", c.action)
	}
}
