package secretx_test

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kilip/omed/api/internal/platform/secretx"
)

func generateKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(key)
}

func TestRoundTrip(t *testing.T) {
	key := generateKey(t)
	c, err := secretx.New(key)
	require.NoError(t, err)
	require.True(t, c.Enabled())

	plaintext := "sk-ant-api03-test-secret-key-12345"
	aad := "7"

	ciphertext, err := c.Encrypt(plaintext, aad)
	require.NoError(t, err)

	decrypted, err := c.Decrypt(ciphertext, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestOutputHasV1Prefix(t *testing.T) {
	key := generateKey(t)
	c, err := secretx.New(key)
	require.NoError(t, err)

	ciphertext, err := c.Encrypt("secret", "42")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(ciphertext, "v1."), "ciphertext should start with v1., got %s", ciphertext)
}

func TestNonceIsRandom(t *testing.T) {
	key := generateKey(t)
	c, err := secretx.New(key)
	require.NoError(t, err)

	ct1, err := c.Encrypt("same-secret", "1")
	require.NoError(t, err)

	ct2, err := c.Encrypt("same-secret", "1")
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "subsequent encryptions of the same plaintext must produce different ciphertexts")
}

func TestWrongAAD(t *testing.T) {
	key := generateKey(t)
	c, err := secretx.New(key)
	require.NoError(t, err)

	ciphertext, err := c.Encrypt("secret", "workspace-1")
	require.NoError(t, err)

	_, err = c.Decrypt(ciphertext, "workspace-2")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDecrypt), "expected ErrDecrypt on wrong AAD, got %v", err)
}

func TestTamperedCiphertext(t *testing.T) {
	key := generateKey(t)
	c, err := secretx.New(key)
	require.NoError(t, err)

	ciphertext, err := c.Encrypt("secret", "7")
	require.NoError(t, err)

	// Ciphertext is "v1.<base64url>"
	parts := strings.SplitN(ciphertext, ".", 2)
	require.Len(t, parts, 2)

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	// Flip a byte in the encoded portion
	raw[len(raw)-1] ^= 0x01
	tampered := "v1." + base64.RawURLEncoding.EncodeToString(raw)

	_, err = c.Decrypt(tampered, "7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDecrypt), "expected ErrDecrypt on tampered ciphertext, got %v", err)
}

func TestWrongKey(t *testing.T) {
	key1 := generateKey(t)
	key2 := generateKey(t)

	c1, err := secretx.New(key1)
	require.NoError(t, err)

	c2, err := secretx.New(key2)
	require.NoError(t, err)

	ciphertext, err := c1.Encrypt("secret", "7")
	require.NoError(t, err)

	_, err = c2.Decrypt(ciphertext, "7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDecrypt), "expected ErrDecrypt on wrong key, got %v", err)
}

func TestUnknownVersionPrefix(t *testing.T) {
	key := generateKey(t)
	c, err := secretx.New(key)
	require.NoError(t, err)

	ciphertext, err := c.Encrypt("secret", "7")
	require.NoError(t, err)

	// Replace "v1." with "v9."
	tampered := "v9." + strings.TrimPrefix(ciphertext, "v1.")
	_, err = c.Decrypt(tampered, "7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDecrypt), "expected ErrDecrypt on unknown version prefix, got %v", err)

	// No version prefix at all
	_, err = c.Decrypt("not-a-valid-ciphertext", "7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDecrypt), "expected ErrDecrypt when missing prefix, got %v", err)
}

func TestEmptyKeyDisabled(t *testing.T) {
	c, err := secretx.New("")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.False(t, c.Enabled())

	_, err = c.Encrypt("secret", "7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDisabled), "expected ErrDisabled from Encrypt, got %v", err)

	_, err = c.Decrypt("v1.dummy", "7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, secretx.ErrDisabled), "expected ErrDisabled from Decrypt, got %v", err)
}

func TestInvalidKey(t *testing.T) {
	testCases := []struct {
		name string
		key  string
	}{
		{
			name: "not base64",
			key:  "this is not base64 encoded!",
		},
		{
			name: "16 bytes base64 (AES-128)",
			key:  base64.StdEncoding.EncodeToString(make([]byte, 16)),
		},
		{
			name: "24 bytes base64 (AES-192)",
			key:  base64.StdEncoding.EncodeToString(make([]byte, 24)),
		},
		{
			name: "64 bytes base64 (too long)",
			key:  base64.StdEncoding.EncodeToString(make([]byte, 64)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := secretx.New(tc.key)
			require.Error(t, err)
			assert.Nil(t, c)
			assert.True(t, errors.Is(err, secretx.ErrInvalidKey), "expected ErrInvalidKey, got %v", err)
		})
	}
}

func TestSupportedBase64Encodings(t *testing.T) {
	rawKey := make([]byte, 32)
	// Fill with distinct bytes to ensure url/std characters differ
	for i := range rawKey {
		rawKey[i] = byte(i * 7)
	}

	encodings := map[string]string{
		"StdEncoding":    base64.StdEncoding.EncodeToString(rawKey),
		"RawStdEncoding": base64.RawStdEncoding.EncodeToString(rawKey),
		"URLEncoding":    base64.URLEncoding.EncodeToString(rawKey),
		"RawURLEncoding": base64.RawURLEncoding.EncodeToString(rawKey),
	}

	for name, encodedKey := range encodings {
		t.Run(name, func(t *testing.T) {
			c, err := secretx.New(encodedKey)
			require.NoError(t, err)
			require.True(t, c.Enabled())

			ct, err := c.Encrypt("test", "1")
			require.NoError(t, err)

			pt, err := c.Decrypt(ct, "1")
			require.NoError(t, err)
			assert.Equal(t, "test", pt)
		})
	}
}
