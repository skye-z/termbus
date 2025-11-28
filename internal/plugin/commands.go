package plugin

import (
	"fmt"
	"time"

	"github.com/termbus/termbus/internal/command"
)

// RegisterCommands registers plugin management commands.
func RegisterCommands(registry *command.CommandRegistry, manager *PluginManager, audit *AuditLogger) error {
	if registry == nil || manager == nil {
		return fmt.Errorf("invalid registry or manager")
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.list",
		Description: "List plugins",
		Category:    "plugin",
		Handler: func(ctx *command.ExecutionContext) error {
			plugins := manager.List()
			if ctx.Output != nil {
				for _, plug := range plugins {
					ctx.Output(fmt.Sprintf("%s\t%s\t%s", plug.ID, plug.Name, plug.Version))
				}
			}
			return nil
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.install",
		Description: "Install plugin",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "source", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("source required")
			}
			_, err := manager.Install(ctx.Args[0])
			return err
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.uninstall",
		Description: "Uninstall plugin",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("id required")
			}
			return manager.Uninstall(ctx.Args[0])
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.enable",
		Description: "Enable plugin",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("id required")
			}
			return manager.Enable(ctx.Args[0])
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.disable",
		Description: "Disable plugin",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("id required")
			}
			return manager.Disable(ctx.Args[0])
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.update",
		Description: "Update plugin",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("id required")
			}
			return manager.Update(ctx.Args[0])
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.info",
		Description: "Plugin info",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("id required")
			}
			plug, err := manager.Info(ctx.Args[0])
			if err == nil && ctx.Output != nil {
				ctx.Output(fmt.Sprintf("ID: %s", plug.ID))
				ctx.Output(fmt.Sprintf("Name: %s", plug.Name))
				ctx.Output(fmt.Sprintf("Version: %s", plug.Version))
				ctx.Output(fmt.Sprintf("Path: %s", plug.Path))
			}
			return err
		},
	}); err != nil {
		return err
	}

	if err := registry.Register(&command.Command{
		Name:        "plugin.reload",
		Description: "Reload plugin",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("id required")
			}
			return manager.Reload(ctx.Args[0])
		},
	}); err != nil {
		return err
	}

	if audit != nil {
		_ = registry.Register(&command.Command{
			Name:        "audit.list",
			Description: "List audit entries",
			Category:    "audit",
			Args:        []command.ArgDefinition{{Name: "plugin_id", Required: false}},
			Handler: func(ctx *command.ExecutionContext) error {
				pluginID := ""
				if len(ctx.Args) > 0 {
					pluginID = ctx.Args[0]
				}
				entries := audit.Query(pluginID, time.Time{}, time.Now())
				if ctx.Output != nil {
					for _, entry := range entries {
						ctx.Output(fmt.Sprintf("%s\t%s\t%s", entry.Timestamp.Format(time.RFC3339), entry.Action, entry.Result))
					}
				}
				return nil
			},
		})

		_ = registry.Register(&command.Command{
			Name:        "audit.search",
			Description: "Search audit entries",
			Category:    "audit",
			Args:        []command.ArgDefinition{{Name: "keyword", Required: true}},
			Handler: func(ctx *command.ExecutionContext) error {
				if len(ctx.Args) == 0 {
					return fmt.Errorf("keyword required")
				}
				entries := audit.Search(ctx.Args[0])
				if ctx.Output != nil {
					for _, entry := range entries {
						ctx.Output(fmt.Sprintf("%s\t%s\t%s", entry.Timestamp.Format(time.RFC3339), entry.Action, entry.Result))
					}
				}
				return nil
			},
		})

		_ = registry.Register(&command.Command{
			Name:        "audit.export",
			Description: "Export audit entries",
			Category:    "audit",
			Args: []command.ArgDefinition{
				{Name: "plugin_id", Required: true},
				{Name: "output", Required: true},
			},
			Handler: func(ctx *command.ExecutionContext) error {
				if len(ctx.Args) < 2 {
					return fmt.Errorf("plugin_id and output required")
				}
				err := audit.Export(ctx.Args[0], "json", ctx.Args[1])
				if err == nil && ctx.Output != nil {
					ctx.Output(fmt.Sprintf("exported to %s", ctx.Args[1]))
				}
				return err
			},
		})
	}

	return nil
}
