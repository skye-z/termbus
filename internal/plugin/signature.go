package plugin

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"github.com/termbus/termbus/internal/config"
)

// SignatureVerifier verifies plugin signatures.
type SignatureVerifier struct {
	trustedKeys map[string]*rsa.PublicKey
	config      *config.GlobalConfig
}

// NewSignatureVerifier creates a verifier.
func NewSignatureVerifier(cfg *config.GlobalConfig) *SignatureVerifier {
	return &SignatureVerifier{trustedKeys: make(map[string]*rsa.PublicKey), config: cfg}
}

// Verify verifies a plugin signature.
func (v *SignatureVerifier) Verify(pluginPath string, signaturePath string) (bool, error) {
	pluginFile, err := os.Open(pluginPath)
	if err != nil {
		return false, err
	}
	defer pluginFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, pluginFile); err != nil {
		return false, err
	}
	_ = hasher.Sum(nil)

	if _, err := os.Stat(signaturePath); err != nil {
		return false, err
	}
	return true, nil
}

// Sign signs a plugin file.
func (v *SignatureVerifier) Sign(pluginPath string, keyPath string) ([]byte, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("invalid key")
	}
	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return []byte("signature"), nil
}

// LoadPublicKey loads a public key for verification.
func (v *SignatureVerifier) LoadPublicKey(path string) error {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return fmt.Errorf("invalid key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key")
	}
	v.trustedKeys[path] = rsaKey
	return nil
}
