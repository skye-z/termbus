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
	plugin := &MySQLPlugin{BasePlugin: api.BasePlugin{Name: "mysql", Version: "1.0.0", Description: "MySQL database management", Author: "termbus"}}
	_ = plugin
}
