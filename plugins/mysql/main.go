package main

import (
	"context"
	"io"

	"termbus/plugin-sdk/pkg/api"
)

type MySQLPlugin struct {
	api.BasePlugin
}

func (p *MySQLPlugin) Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 0, nil
}

func (p *MySQLPlugin) Permissions() []string {
	return []string{"system.network"}
}

func (p *MySQLPlugin) Commands() []string {
	return []string{"mysql.query", "mysql.schema", "mysql.tables"}
}

func main() {
	plugin := &MySQLPlugin{BasePlugin: api.BasePlugin{NameValue: "mysql", VersionValue: "1.0.0", DescriptionValue: "MySQL database management", AuthorValue: "termbus"}}
	api.Serve(plugin)
}
