package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrInvalidKey        = errors.New("invalid key")
	ErrEmptyPassword     = errors.New("empty password")
)

const (
	KeySize   = 32
	NonceSize = 12
	SaltSize  = 16
)

// Encryptor 加密器
type Encryptor struct {
	key []byte
}

// NewEncryptor 创建加密器
func NewEncryptor(password string, salt []byte) (*Encryptor, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	if len(salt) == 0 {
		salt = make([]byte, SaltSize)
		if _, err := rand.Read(salt); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	key := deriveKey(password, salt)

	return &Encryptor{
		key: key,
	}, nil
}

// deriveKey 从密码派生密钥
func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, 100000, KeySize, sha256.New)
}

// Encrypt 加密数据
func (e *Encryptor) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	result := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt 解密数据
func (e *Encryptor) Decrypt(ciphertext string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	if len(data) < NonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce := data[:NonceSize]
	ciphertextData := data[NonceSize:]

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	return plaintext, nil
}

// EncryptString 加密字符串
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString 解密字符串
func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateSalt 生成随机盐
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// EncryptWithSalt 使用指定盐加密
func EncryptWithSalt(password string, salt, plaintext []byte) (string, error) {
	encryptor, err := NewEncryptor(password, salt)
	if err != nil {
		return "", err
	}
	return encryptor.Encrypt(plaintext)
}

// DecryptWithSalt 使用指定盐解密
func DecryptWithSalt(password string, salt []byte, ciphertext string) ([]byte, error) {
	encryptor, err := NewEncryptor(password, salt)
	if err != nil {
		return nil, err
	}
	return encryptor.Decrypt(ciphertext)
}

// SecureStorage 安全存储
type SecureStorage struct {
	encryptor *Encryptor
	salt      []byte
}

// NewSecureStorage 创建安全存储
func NewSecureStorage(password string) (*SecureStorage, error) {
	if password == "" {
		return nil, ErrEmptyPassword
	}

	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	encryptor, err := NewEncryptor(password, salt)
	if err != nil {
		return nil, err
	}

	return &SecureStorage{
		encryptor: encryptor,
		salt:      salt,
	}, nil
}

// NewSecureStorageFromSalt 从已知盐创建安全存储
func NewSecureStorageFromSalt(password string, salt []byte) (*SecureStorage, error) {
	if len(salt) != SaltSize {
		return nil, ErrInvalidKey
	}

	encryptor, err := NewEncryptor(password, salt)
	if err != nil {
		return nil, err
	}

	return &SecureStorage{
		encryptor: encryptor,
		salt:      salt,
	}, nil
}

// Store 存储数据
func (s *SecureStorage) Store(data []byte) (string, error) {
	return s.encryptor.Encrypt(data)
}

// Retrieve 获取数据
func (s *SecureStorage) Retrieve(ciphertext string) ([]byte, error) {
	return s.encryptor.Decrypt(ciphertext)
}

// GetSalt 获取盐
func (s *SecureStorage) GetSalt() []byte {
	return s.salt
}

// HashPassword 哈希密码
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash string) bool {
	return HashPassword(password) == hash
}
