package main

import (
	"context"
	"io"

	"termbus/plugin-sdk/pkg/api"
)

type RedisPlugin struct {
	api.BasePlugin
}

func (p *RedisPlugin) Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 0, nil
}

func (p *RedisPlugin) Permissions() []string {
	return []string{"system.network"}
}

func (p *RedisPlugin) Commands() []string {
	return []string{"redis.get", "redis.set", "redis.del", "redis.keys", "redis.info"}
}

func main() {
	plugin := &RedisPlugin{BasePlugin: api.BasePlugin{Name: "redis", Version: "1.0.0", Description: "Redis database management", Author: "termbus"}}
	_ = plugin
}
