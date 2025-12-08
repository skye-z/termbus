package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// GlobalConfig 全局配置
type GlobalConfig struct {
	General GeneralConfig `mapstructure:"general"`
	Log     LogConfig     `mapstructure:"log"`
	SSH     SSHConfig     `mapstructure:"ssh"`
	Plugin  PluginConfig  `mapstructure:"plugin"`
}

// GeneralConfig 通用配置
type GeneralConfig struct {
	ConfigDir     string `mapstructure:"config_dir"`
	DataDir       string `mapstructure:"data_dir"`
	TerminalTheme string `mapstructure:"terminal_theme" default:"default"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level" default:"info"`
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int    `mapstructure:"max_size" default:"100"`
	MaxBackups int    `mapstructure:"max_backups" default:"3"`
	MaxAge     int    `mapstructure:"max_age" default:"7"`
}

// SSHConfig SSH配置
type SSHConfig struct {
	ConfigPath        string   `mapstructure:"config_path"`
	KnownHostsPath    string   `mapstructure:"known_hosts_path"`
	IdentityFiles     []string `mapstructure:"identity_files"`
	DefaultTimeout    int      `mapstructure:"default_timeout" default:"30"`
	KeepaliveEnabled  bool     `mapstructure:"keepalive_enabled" default:"true"`
	KeepaliveInterval int      `mapstructure:"keepalive_interval" default:"60"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	PluginDir string            `mapstructure:"plugin_dir"`
	Enabled   map[string]bool   `mapstructure:"enabled"`
	Options   map[string]string `mapstructure:"options"`
}

// Manager 配置管理器
type Manager struct {
	config      *GlobalConfig
	v           *viper.Viper
	configPath  string
	mu          sync.RWMutex
	hotReloadCh chan struct{}
}

// New 创建配置管理器
func New() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".termbus")
	configPath := filepath.Join(configDir, "config.yaml")

	m := &Manager{
		v:           viper.New(),
		configPath:  configPath,
		hotReloadCh: make(chan struct{}, 1),
	}

	m.v.SetConfigFile(configPath)
	m.v.SetConfigType("yaml")

	m.v.SetDefault("general.config_dir", configDir)
	m.v.SetDefault("general.data_dir", filepath.Join(configDir, "data"))
	m.v.SetDefault("log.output_path", filepath.Join(configDir, "logs", "core.log"))
	m.v.SetDefault("log.level", "info")
	m.v.SetDefault("ssh.config_path", filepath.Join(homeDir, ".ssh", "config"))
	m.v.SetDefault("ssh.known_hosts_path", filepath.Join(homeDir, ".ssh", "known_hosts"))
	m.v.SetDefault("ssh.default_timeout", 30)
	m.v.SetDefault("ssh.keepalive_enabled", true)
	m.v.SetDefault("ssh.keepalive_interval", 60)
	m.v.SetDefault("plugin.plugin_dir", filepath.Join(configDir, "plugins"))

	if err := m.load(); err != nil {
		return nil, err
	}

	return m, nil
}

// Load 加载配置
func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	configDir := filepath.Dir(m.configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		if err := m.v.WriteConfigAs(m.configPath); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}
	}

	if err := m.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config GlobalConfig
	if err := m.v.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	m.config = &config
	return nil
}

// Reload 重新加载配置
func (m *Manager) Reload() error {
	if err := m.load(); err != nil {
		return err
	}
	select {
	case m.hotReloadCh <- struct{}{}:
	default:
	}
	return nil
}

// Get 获取配置
func (m *Manager) Get() *GlobalConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetViper 获取viper实例
func (m *Manager) GetViper() *viper.Viper {
	return m.v
}

// GetHotReloadCh 获取热加载通道
func (m *Manager) GetHotReloadCh() <-chan struct{} {
	return m.hotReloadCh
}
