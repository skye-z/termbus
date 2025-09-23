package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name     string
		password string
		salt     []byte
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "test-password",
			salt:     []byte("test-salt-123456"),
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			salt:     []byte("test-salt-123456"),
			wantErr:  true,
		},
		{
			name:     "no salt",
			password: "test-password",
			salt:     nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewEncryptor(tt.password, tt.salt)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
				assert.NotNil(t, encryptor.key)
			}
		})
	}
}

func TestEncryptorEncryptDecrypt(t *testing.T) {
	password := "test-password"
	salt := make([]byte, SaltSize)

	encryptor, err := NewEncryptor(password, salt)
	require.NoError(t, err)

	plaintext := []byte("Hello, World!")

	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, string(plaintext), ciphertext)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, SaltSize)

	salt2, err := GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt2, SaltSize)

	assert.NotEqual(t, salt1, salt2)
}

func TestHashPassword(t *testing.T) {
	password := "test-password"

	hash1 := HashPassword(password)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64)

	hash2 := HashPassword(password)
	assert.Equal(t, hash1, hash2)
}

func TestVerifyPassword(t *testing.T) {
	password := "test-password"

	hash := HashPassword(password)

	assert.True(t, VerifyPassword(password, hash))
	assert.False(t, VerifyPassword("wrong-password", hash))
}
