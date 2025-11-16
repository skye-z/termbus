package plugin

import "testing"

func TestRuntimeLoad(t *testing.T) {
	runtime := NewRuntime(nil, nil)
	plugin, err := runtime.Load("/tmp/plugin")
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if plugin.Path == "" {
		t.Fatalf("expected path")
	}
}
