package security

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

// KeyEntry 密钥条目
type KeyEntry struct {
	Key        string `json:"key"`
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
}

// Keyring 密钥环
type Keyring struct {
	dataDir  string
	storage  *SecureStorage
	entries  map[string]*KeyEntry
	password string
}

// NewKeyring 创建密钥环
func NewKeyring(dataDir, password string) (*Keyring, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	storage, err := NewSecureStorage(password)
	if err != nil {
		return nil, err
	}

	keyring := &Keyring{
		dataDir:  dataDir,
		storage:  storage,
		entries:  make(map[string]*KeyEntry),
		password: password,
	}

	if err := keyring.load(); err != nil {
		logger.GetLogger().Warn("Failed to load keyring",
			zap.Any("error", err),
		)
	}

	return keyring, nil
}

// Set 设置密钥
func (k *Keyring) Set(key, plaintext string) error {
	ciphertext, err := k.storage.Store([]byte(plaintext))
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	entry := &KeyEntry{
		Key:        key,
		Ciphertext: ciphertext,
		Salt:       fmt.Sprintf("%x", k.storage.GetSalt()),
	}

	k.entries[key] = entry

	if err := k.save(); err != nil {
		return fmt.Errorf("failed to save keyring: %w", err)
	}

	return nil
}

// Get 获取密钥
func (k *Keyring) Get(key string) (string, error) {
	entry, exists := k.entries[key]
	if !exists {
		return "", fmt.Errorf("key not found: %s", key)
	}

	salt, err := decodeHex(entry.Salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	storage, err := NewSecureStorageFromSalt(k.password, salt)
	if err != nil {
		return "", fmt.Errorf("failed to create storage: %w", err)
	}

	plaintext, err := storage.Retrieve(entry.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// Delete 删除密钥
func (k *Keyring) Delete(key string) error {
	if _, exists := k.entries[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(k.entries, key)

	if err := k.save(); err != nil {
		return fmt.Errorf("failed to save keyring: %w", err)
	}

	return nil
}

// Exists 检查密钥是否存在
func (k *Keyring) Exists(key string) bool {
	_, exists := k.entries[key]
	return exists
}

// List 列出所有密钥
func (k *Keyring) List() []string {
	keys := make([]string, 0, len(k.entries))
	for key := range k.entries {
		keys = append(keys, key)
	}
	return keys
}

// Load 加载密钥环
func (k *Keyring) load() error {
	keyringFile := k.getKeyringFilePath()

	data, err := os.ReadFile(keyringFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read keyring file: %w", err)
	}

	var entries []KeyEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal keyring: %w", err)
	}

	for i := range entries {
		k.entries[entries[i].Key] = &entries[i]
	}

	logger.GetLogger().Info("Keyring loaded",
		zap.Int("count", len(k.entries)),
	)

	return nil
}

// Save 保存密钥环
func (k *Keyring) save() error {
	keyringFile := k.getKeyringFilePath()

	entries := make([]KeyEntry, 0, len(k.entries))
	for _, entry := range k.entries {
		entries = append(entries, *entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keyring: %w", err)
	}

	if err := os.WriteFile(keyringFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write keyring file: %w", err)
	}

	return nil
}

// getKeyringFilePath 获取密钥环文件路径
func (k *Keyring) getKeyringFilePath() string {
	return filepath.Join(k.dataDir, "keyring.json")
}

// decodeHex 解码十六进制字符串
func decodeHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// Clear 清空密钥环
func (k *Keyring) Clear() error {
	k.entries = make(map[string]*KeyEntry)
	return k.save()
}
