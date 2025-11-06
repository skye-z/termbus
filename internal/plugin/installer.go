package plugin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/termbus/termbus/internal/config"
)

// PluginInstaller installs and verifies plugins.
type PluginInstaller struct {
	installDir string
	config     *config.GlobalConfig
}

// NewInstaller creates a plugin installer.
func NewInstaller(installDir string, cfg *config.GlobalConfig) *PluginInstaller {
	return &PluginInstaller{installDir: installDir, config: cfg}
}

// InstallFromURL installs plugin from a URL.
func (i *PluginInstaller) InstallFromURL(url string) (*Plugin, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad response: %s", resp.Status)
	}

	fileName := filepath.Base(url)
	if fileName == "" {
		return nil, fmt.Errorf("invalid url")
	}

	if err := os.MkdirAll(i.installDir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(i.installDir, fileName)
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return nil, err
	}

	return &Plugin{Path: path}, nil
}

// InstallFromFile installs plugin from file.
func (i *PluginInstaller) InstallFromFile(file string) (*Plugin, error) {
	if err := os.MkdirAll(i.installDir, 0755); err != nil {
		return nil, err
	}
	base := filepath.Base(file)
	path := filepath.Join(i.installDir, base)

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0755); err != nil {
		return nil, err
	}
	return &Plugin{Path: path}, nil
}

// InstallFromDir installs plugin from a directory.
func (i *PluginInstaller) InstallFromDir(dir string) (*Plugin, error) {
	bin := filepath.Join(dir, filepath.Base(dir))
	if _, err := os.Stat(bin); err != nil {
		return nil, err
	}
	return &Plugin{Path: bin}, nil
}

// Verify verifies plugin before install.
func (i *PluginInstaller) Verify(plugin *Plugin) error {
	if plugin == nil || plugin.Path == "" {
		return fmt.Errorf("invalid plugin")
	}
	return nil
}

// Uninstall removes a plugin.
func (i *PluginInstaller) Uninstall(id string) error {
	if id == "" {
		return fmt.Errorf("invalid plugin id")
	}
	return os.Remove(id)
}
