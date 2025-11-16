package plugin

import "testing"

func TestPermissionGrant(t *testing.T) {
	manager := NewPermissionManager(nil)
	if manager.Check("p1", PermSSHExecute) {
		t.Fatalf("permission should not be granted")
	}
	_ = manager.Grant("p1", PermSSHExecute)
	if !manager.Check("p1", PermSSHExecute) {
		t.Fatalf("permission should be granted")
	}
}
