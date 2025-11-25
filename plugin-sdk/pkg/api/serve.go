package api

import (
	"context"
	"io"

	"github.com/termbus/termbus/pkg/pluginrpc"
)

// Serve runs the plugin RPC server.
func Serve(p Plugin) {
	pluginrpc.Serve(&adapter{plugin: p})
}

type adapter struct {
	plugin Plugin
}

func (a *adapter) Init(ctx context.Context, sessionID string, config map[string]string) error {
	_ = sessionID
	return a.plugin.Init(ctx, config)
}

func (a *adapter) Execute(ctx context.Context, command string, args []string, env map[string]string) (*pluginrpc.ExecuteResponse, error) {
	_, _ = env, ctx
	stdout := io.Discard
	stderr := io.Discard
	code, err := a.plugin.Execute(ctx, command, args, nil, stdout, stderr)
	resp := &pluginrpc.ExecuteResponse{ExitCode: code}
	if err != nil {
		resp.Error = err.Error()
	}
	return resp, nil
}

func (a *adapter) Stop(ctx context.Context, force bool) error {
	return a.plugin.Stop(ctx)
}

func (a *adapter) Info(ctx context.Context) (*pluginrpc.InfoResponse, error) {
	return &pluginrpc.InfoResponse{
		Name:        a.plugin.Name(),
		Version:     a.plugin.Version(),
		Description: a.plugin.Description(),
		Author:      a.plugin.Author(),
	}, nil
}

func (a *adapter) Manifest(ctx context.Context) (*pluginrpc.ManifestResponse, error) {
	return &pluginrpc.ManifestResponse{
		Name:        a.plugin.Name(),
		Version:     a.plugin.Version(),
		Description: a.plugin.Description(),
		Author:      a.plugin.Author(),
		Permissions: a.plugin.Permissions(),
		Commands:    a.plugin.Commands(),
	}, nil
}
