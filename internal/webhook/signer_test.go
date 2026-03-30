package webhook_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/guilhermecosta/wpp-gateway/internal/webhook"
)

func TestSignAndVerify(t *testing.T) {
	payload := []byte(`{"event":"test","data":{}}`)
	secret := "my-secret-key"

	signature := webhook.Sign(payload, secret)
	assert.NotEmpty(t, signature)
	assert.True(t, len(signature) > 7, "should have sha256= prefix + hash")

	assert.True(t, webhook.Verify(payload, secret, signature))
}

func TestVerifyWrongSecret(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	sig := webhook.Sign(payload, "correct-secret")

	assert.False(t, webhook.Verify(payload, "wrong-secret", sig))
}

func TestVerifyTamperedPayload(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	secret := "my-secret"
	sig := webhook.Sign(payload, secret)

	tampered := []byte(`{"event":"hacked"}`)
	assert.False(t, webhook.Verify(tampered, secret, sig))
}

func TestSignDeterministic(t *testing.T) {
	payload := []byte("same payload")
	secret := "same secret"

	sig1 := webhook.Sign(payload, secret)
	sig2 := webhook.Sign(payload, secret)
	assert.Equal(t, sig1, sig2)
}

func TestSignDifferentSecrets(t *testing.T) {
	payload := []byte("payload")
	sig1 := webhook.Sign(payload, "secret1")
	sig2 := webhook.Sign(payload, "secret2")
	assert.NotEqual(t, sig1, sig2)
}
