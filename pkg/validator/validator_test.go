package validator_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guilhermecosta/wpp-gateway/pkg/validator"
)

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"valid BR mobile", "5511999999999", false},
		{"valid 10 digits", "1234567890", false},
		{"valid 15 digits", "123456789012345", false},
		{"too short", "123456789", true},
		{"too long", "1234567890123456", true},
		{"has letters", "551199abc9999", true},
		{"has plus", "+5511999999999", true},
		{"empty", "", true},
		{"has spaces", "55 11 99999", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePhone(tt.phone)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	validUUID := uuid.New().String()
	parsed, err := validator.ValidateUUID(validUUID)
	require.NoError(t, err)
	assert.Equal(t, validUUID, parsed.String())

	_, err = validator.ValidateUUID("not-a-uuid")
	assert.Error(t, err)

	_, err = validator.ValidateUUID("")
	assert.Error(t, err)
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://localhost:3000/hook", false},
		{"no scheme", "example.com/webhook", true},
		{"ftp scheme", "ftp://example.com/hook", true},
		{"empty", "", true},
		{"just path", "/webhook", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWebhookURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
