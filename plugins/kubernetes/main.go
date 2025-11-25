package main

import (
	"context"
	"io"

	"termbus/plugin-sdk/pkg/api"
)

type KubernetesPlugin struct {
	api.BasePlugin
}

func (p *KubernetesPlugin) Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	return 0, nil
}

func (p *KubernetesPlugin) Permissions() []string {
	return []string{"ssh.execute", "system.network"}
}

func (p *KubernetesPlugin) Commands() []string {
	return []string{"k8s.pods", "k8s.deploy", "k8s.logs", "k8s.exec"}
}

func main() {
	plugin := &KubernetesPlugin{BasePlugin: api.BasePlugin{NameValue: "kubernetes", VersionValue: "1.0.0", DescriptionValue: "Kubernetes cluster management", AuthorValue: "termbus"}}
	api.Serve(plugin)
}
