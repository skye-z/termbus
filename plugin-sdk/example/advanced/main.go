package main

import (
	"context"
	"io"

	"termbus/plugin-sdk/pkg/api"
)

type AdvancedPlugin struct {
	api.BasePlugin
}

func (p *AdvancedPlugin) Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 0, nil
}

func (p *AdvancedPlugin) Permissions() []string {
	return []string{"system.network"}
}

func (p *AdvancedPlugin) Commands() []string {
	return []string{"advanced"}
}

func main() {
	plugin := &AdvancedPlugin{
		BasePlugin: api.BasePlugin{
			NameValue:        "advanced",
			VersionValue:     "0.1.0",
			DescriptionValue: "Advanced plugin example",
			AuthorValue:      "termbus",
		},
	}

	api.Serve(plugin)
}
