package config

import (
	"fmt"

	"github.com/termbus/termbus/internal/security"
)

// SecureConfigManager 安全配置管理器
type SecureConfigManager struct {
	*Manager
	keyring *security.Keyring
}

// NewSecureConfigManager 创建安全配置管理器
func NewSecureConfigManager(password string) (*SecureConfigManager, error) {
	manager, err := New()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	keyring, err := security.NewKeyring(manager.Get().General.DataDir, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	return &SecureConfigManager{
		Manager: manager,
		keyring: keyring,
	}, nil
}

// SetSecureValue 设置安全值（加密存储）
func (m *SecureConfigManager) SetSecureValue(key, value string) error {
	return m.keyring.Set(key, value)
}

// GetSecureValue 获取安全值（解密）
func (m *SecureConfigManager) GetSecureValue(key string) (string, error) {
	return m.keyring.Get(key)
}

// DeleteSecureValue 删除安全值
func (m *SecureConfigManager) DeleteSecureValue(key string) error {
	return m.keyring.Delete(key)
}

// ExistsSecureValue 检查安全值是否存在
func (m *SecureConfigManager) ExistsSecureValue(key string) bool {
	return m.keyring.Exists(key)
}

// ListSecureKeys 列出所有安全密钥
func (m *SecureConfigManager) ListSecureKeys() []string {
	return m.keyring.List()
}

// SetHostPassword 设置主机密码
func (m *SecureConfigManager) SetHostPassword(host, password string) error {
	key := fmt.Sprintf("host.%s.password", host)
	return m.SetSecureValue(key, password)
}

// GetHostPassword 获取主机密码
func (m *SecureConfigManager) GetHostPassword(host string) (string, error) {
	key := fmt.Sprintf("host.%s.password", host)
	return m.GetSecureValue(key)
}

// DeleteHostPassword 删除主机密码
func (m *SecureConfigManager) DeleteHostPassword(host string) error {
	key := fmt.Sprintf("host.%s.password", host)
	return m.DeleteSecureValue(key)
}

// SetAPIKey 设置API密钥
func (m *SecureConfigManager) SetAPIKey(service, apiKey string) error {
	key := fmt.Sprintf("api.%s.key", service)
	return m.SetSecureValue(key, apiKey)
}

// GetAPIKey 获取API密钥
func (m *SecureConfigManager) GetAPIKey(service string) (string, error) {
	key := fmt.Sprintf("api.%s.key", service)
	return m.GetSecureValue(key)
}
