package secretx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

var (
	ErrDisabled   = errors.New("secretx: cipher disabled")
	ErrInvalidKey = errors.New("secretx: invalid master key")
	ErrDecrypt    = errors.New("secretx: decryption failed")
)

const versionPrefix = "v1."

type Cipher struct {
	enabled bool
	aead    cipher.AEAD
}

// New creates a new Cipher from a base64-encoded 32-byte master key.
// If masterKeyB64 is empty, a disabled Cipher is returned with nil error.
func New(masterKeyB64 string) (*Cipher, error) {
	if masterKeyB64 == "" {
		return &Cipher{enabled: false}, nil
	}

	key, err := decodeKey(masterKeyB64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Cipher{
		enabled: true,
		aead:    gcm,
	}, nil
}

// Enabled returns true if the cipher is initialized with a valid master key.
func (c *Cipher) Enabled() bool {
	return c.enabled
}

// Encrypt encrypts plaintext with AES-256-GCM and workspace AAD,
// returning "v1." + base64.RawURLEncoding(nonce || ciphertext || tag).
func (c *Cipher) Encrypt(plaintext, aad string) (string, error) {
	if !c.enabled {
		return "", ErrDisabled
	}

	nonce := make([]byte, c.aead.NonceSize(), c.aead.NonceSize()+len(plaintext)+c.aead.Overhead())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), []byte(aad))
	return versionPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt verifies the "v1." prefix, base64url-decodes the payload, and decrypts with AES-256-GCM and AAD.
func (c *Cipher) Decrypt(stored, aad string) (string, error) {
	if !c.enabled {
		return "", ErrDisabled
	}

	if !strings.HasPrefix(stored, versionPrefix) {
		return "", ErrDecrypt
	}

	raw, err := base64.RawURLEncoding.DecodeString(stored[len(versionPrefix):])
	if err != nil {
		return "", ErrDecrypt
	}

	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", ErrDecrypt
	}

	nonce := raw[:nonceSize]
	ciphertext := raw[nonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return "", ErrDecrypt
	}

	return string(plaintext), nil
}

func decodeKey(s string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, enc := range encodings {
		b, err := enc.DecodeString(s)
		if err == nil {
			if len(b) != 32 {
				return nil, ErrInvalidKey
			}
			return b, nil
		}
	}

	return nil, ErrInvalidKey
}
