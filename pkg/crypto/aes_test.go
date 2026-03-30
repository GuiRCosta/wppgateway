package crypto_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guilhermecosta/wpp-gateway/pkg/crypto"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("hello world, this is a secret message")

	ciphertext, nonce, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEmpty(t, nonce)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := crypto.Decrypt(ciphertext, nonce, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("same message")

	ct1, _, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)

	ct2, _, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "two encryptions of same plaintext should produce different ciphertexts")
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)
	plaintext := []byte("secret")

	ciphertext, nonce, err := crypto.Encrypt(plaintext, key1)
	require.NoError(t, err)

	_, err = crypto.Decrypt(ciphertext, nonce, key2)
	assert.Error(t, err)
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := testKey(t)
	plaintext := []byte("do not tamper")

	ciphertext, nonce, err := crypto.Encrypt(plaintext, key)
	require.NoError(t, err)

	ciphertext[0] ^= 0xFF

	_, err = crypto.Decrypt(ciphertext, nonce, key)
	assert.Error(t, err)
}

func TestParseKeyValid(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key, err := crypto.ParseKey(hexKey)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestParseKeyInvalidLength(t *testing.T) {
	_, err := crypto.ParseKey("0123456789abcdef")
	assert.Error(t, err)
}

func TestParseKeyInvalidHex(t *testing.T) {
	_, err := crypto.ParseKey("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	assert.Error(t, err)
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := testKey(t)
	ciphertext, nonce, err := crypto.Encrypt([]byte{}, key)
	require.NoError(t, err)

	decrypted, err := crypto.Decrypt(ciphertext, nonce, key)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}
