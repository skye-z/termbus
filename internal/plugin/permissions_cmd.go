package plugin

import (
	"fmt"

	"github.com/termbus/termbus/internal/command"
)

// RegisterPermissionCommands registers permission commands.
func RegisterPermissionCommands(registry *command.CommandRegistry, authorizer *Authorizer) error {
	if registry == nil || authorizer == nil {
		return fmt.Errorf("invalid registry or authorizer")
	}
	if err := registry.Register(&command.Command{
		Name:        "plugin.permission.revoke",
		Description: "Revoke plugin permission",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "plugin_id", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("plugin_id required")
			}
			err := authorizer.RevokeAuthorization(ctx.Args[0])
			if err == nil && ctx.Output != nil {
				ctx.Output("permission revoked")
			}
			return err
		},
	}); err != nil {
		return err
	}

	return registry.Register(&command.Command{
		Name:        "plugin.permission.request",
		Description: "Request plugin permission",
		Category:    "plugin",
		Args:        []command.ArgDefinition{{Name: "plugin_id", Required: true}, {Name: "permission", Required: true}},
		Handler: func(ctx *command.ExecutionContext) error {
			if len(ctx.Args) < 2 {
				return fmt.Errorf("plugin_id and permission required")
			}
			_, err := authorizer.RequestAuthorization(&AuthorizationRequest{PluginID: ctx.Args[0], Permissions: []Permission{Permission(ctx.Args[1])}})
			if err == nil && ctx.Output != nil {
				ctx.Output("permission requested")
			}
			return err
		},
	})
}
