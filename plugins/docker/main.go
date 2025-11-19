package main

import (
	"context"
	"io"

	"termbus/plugin-sdk/pkg/api"
)

type DockerPlugin struct {
	api.BasePlugin
}

func (p *DockerPlugin) Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 0, nil
}

func (p *DockerPlugin) Permissions() []string {
	return []string{"ssh.execute", "system.exec"}
}

func (p *DockerPlugin) Commands() []string {
	return []string{"docker.ps", "docker.logs", "docker.exec", "docker.images"}
}

func main() {
	plugin := &DockerPlugin{BasePlugin: api.BasePlugin{Name: "docker", Version: "1.0.0", Description: "Docker container management", Author: "termbus"}}
	_ = plugin
}
