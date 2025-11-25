package main

import (
	"context"
	"fmt"
	"io"

	"termbus/plugin-sdk/pkg/api"
)

type HelloPlugin struct {
	api.BasePlugin
}

func (p *HelloPlugin) Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	fmt.Fprintf(stdout, "Hello from Termbus Plugin!\n")
	fmt.Fprintf(stdout, "Command: %s, Args: %v\n", cmd, args)
	return 0, nil
}

func (p *HelloPlugin) Permissions() []string {
	return []string{"ssh.execute"}
}

func (p *HelloPlugin) Commands() []string {
	return []string{"hello"}
}

func main() {
	plugin := &HelloPlugin{
		BasePlugin: api.BasePlugin{
			NameValue:        "hello",
			VersionValue:     "1.0.0",
			DescriptionValue: "A simple hello world plugin",
			AuthorValue:      "termbus",
		},
	}

	api.Serve(plugin)
}
