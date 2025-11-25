package plugin

import (
	"fmt"

	"github.com/termbus/termbus/internal/command"
)

// RegisterCommands registers plugin management commands.
func RegisterCommands(registry *command.CommandRegistry, manager *PluginManager) error {
	if registry == nil || manager == nil {
		return fmt.Errorf("invalid registry or manager")
	}

	return registry.Register(&command.Command{
		Name:        "plugin.list",
		Description: "List plugins",
		Category:    "plugin",
		Handler: func(ctx *command.ExecutionContext) error {
			_ = manager.List()
			return nil
		},
	})
}
