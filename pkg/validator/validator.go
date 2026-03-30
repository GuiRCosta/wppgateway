package validator

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/google/uuid"
)

var phoneRegex = regexp.MustCompile(`^\d{10,15}$`)

func ValidatePhone(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number: must be 10-15 digits")
	}
	return nil
}

func ValidateUUID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID: %s", id)
	}
	return parsed, nil
}

func ValidateWebhookURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use http or https scheme")
	}
	return nil
}
