package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/plugin"
)

func TestE2E_PluginStore(t *testing.T) {
	store := plugin.NewStore(nil)

	testPlugin := &plugin.Plugin{
		ID:      "test-plugin",
		Name:    "Test Plugin",
		Version: "1.0.0",
		Path:    "/tmp/test-plugin",
		Enabled: true,
	}

	err := store.Add(testPlugin)
	require.NoError(t, err)

	p, err := store.Get("/tmp/test-plugin")
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", p.ID)

	list := store.List()
	assert.Equal(t, 1, len(list))
}

func TestE2E_PluginPermission(t *testing.T) {
	permMgr := plugin.NewPermissionManager(nil)

	testPerm := plugin.PermSSHExecute

	err := permMgr.Grant("test-plugin", testPerm)
	require.NoError(t, err)

	hasPerm := permMgr.Check("test-plugin", testPerm)
	assert.True(t, hasPerm)

	perms := permMgr.List("test-plugin")
	assert.Equal(t, 1, len(perms))

	err = permMgr.Revoke("test-plugin", testPerm)
	require.NoError(t, err)

	hasPerm = permMgr.Check("test-plugin", testPerm)
	assert.False(t, hasPerm)
}

func TestE2E_PluginPermissionMultiple(t *testing.T) {
	permMgr := plugin.NewPermissionManager(nil)

	perms := []plugin.Permission{plugin.PermSSHExecute, plugin.PermSFTPRead, plugin.PermSFTPWrite}

	for _, p := range perms {
		err := permMgr.Grant("test-plugin", p)
		require.NoError(t, err)
	}

	list := permMgr.List("test-plugin")
	assert.Equal(t, 3, len(list))

	assert.True(t, permMgr.Check("test-plugin", plugin.PermSSHExecute))
	assert.True(t, permMgr.Check("test-plugin", plugin.PermSFTPRead))
	assert.True(t, permMgr.Check("test-plugin", plugin.PermSFTPWrite))
	assert.False(t, permMgr.Check("test-plugin", plugin.PermSFTPDelete))
}

func TestE2E_PluginPermissionRequest(t *testing.T) {
	permMgr := plugin.NewPermissionManager(nil)

	err := permMgr.Request("test-plugin", plugin.PermSystemExec)
	require.NoError(t, err)

	perms := permMgr.List("test-plugin")
	assert.Equal(t, 1, len(perms))
}

func TestE2E_PluginRuntime(t *testing.T) {
	runtime := plugin.NewRuntime(nil, nil)

	plugins := runtime.List()
	assert.Equal(t, 0, len(plugins))

	plugins = runtime.ListEnabled()
	assert.Equal(t, 0, len(plugins))

	_, err := runtime.Get("nonexistent")
	assert.Error(t, err)
}

func TestE2E_PluginManifest(t *testing.T) {
	manifest := &plugin.PluginManifest{
		Name:        "Test Manifest",
		Version:     "1.0.0",
		Description: "Test manifest description",
		Author:      "Test Author",
		Commands:    []string{"cmd1", "cmd2"},
		Permissions: []string{"exec", "file.read"},
	}

	assert.Equal(t, "Test Manifest", manifest.Name)
	assert.Equal(t, 2, len(manifest.Commands))
	assert.Equal(t, 2, len(manifest.Permissions))
}

func TestE2E_PluginLoader(t *testing.T) {
	loader := plugin.NewLoader(nil, nil)

	assert.NotNil(t, loader)
}

func TestE2E_PluginAuthorizer(t *testing.T) {
	permMgr := plugin.NewPermissionManager(nil)
	authorizer := plugin.NewAuthorizer(permMgr, nil, "")

	req := &plugin.AuthorizationRequest{
		PluginID:    "test-plugin",
		Permissions: []plugin.Permission{plugin.PermSSHConnect},
		Reason:      "Testing authorization",
	}

	decision, err := authorizer.RequestAuthorization(req)
	require.NoError(t, err)
	assert.NotNil(t, decision)

	authorizer.GrantAll("test-plugin")
	authorizer.RevokeAuthorization("test-plugin")

	auths := authorizer.ListAuthorizations("test-plugin")
	assert.Equal(t, 0, len(auths))
}

func TestE2E_PluginInstaller(t *testing.T) {
	installer := plugin.NewInstaller("/tmp", nil)

	assert.NotNil(t, installer)
}

func TestE2E_PluginSignatureVerifier(t *testing.T) {
	verifier := plugin.NewSignatureVerifier(nil)

	assert.NotNil(t, verifier)

	_, err := verifier.Verify("/nonexistent/plugin", "/nonexistent/sig")
	assert.Error(t, err)
}

func TestE2E_PluginPermissionConstants(t *testing.T) {
	assert.Equal(t, plugin.Permission("ssh.connect"), plugin.PermSSHConnect)
	assert.Equal(t, plugin.Permission("ssh.execute"), plugin.PermSSHExecute)
	assert.Equal(t, plugin.Permission("sftp.read"), plugin.PermSFTPRead)
	assert.Equal(t, plugin.Permission("sftp.write"), plugin.PermSFTPWrite)
	assert.Equal(t, plugin.Permission("sftp.delete"), plugin.PermSFTPDelete)
	assert.Equal(t, plugin.Permission("tunnel.create"), plugin.PermTunnelCreate)
	assert.Equal(t, plugin.Permission("tunnel.manage"), plugin.PermTunnelManage)
	assert.Equal(t, plugin.Permission("config.read"), plugin.PermConfigRead)
	assert.Equal(t, plugin.Permission("config.write"), plugin.PermConfigWrite)
	assert.Equal(t, plugin.Permission("system.exec"), plugin.PermSystemExec)
	assert.Equal(t, plugin.Permission("system.network"), plugin.PermSystemNetwork)
}
