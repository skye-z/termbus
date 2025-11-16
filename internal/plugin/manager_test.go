package plugin

import "testing"

func TestManagerList(t *testing.T) {
	runtime := NewRuntime(nil, nil)
	manager := NewManager(runtime, nil, nil, nil, nil)
	if len(manager.List()) != 0 {
		t.Fatalf("expected empty list")
	}
}
