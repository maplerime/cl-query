/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"strings"
	"time"
	"web/src/model"

	"golang.org/x/crypto/bcrypt"

	. "github.com/maplerime/cl-query/pkg/common"
)

type APIKeyAdmin struct{}

// ParseAPIKey splits a cl_<uuid>_<secret> key into its UUID and secret
// components. The UUID is always 36 chars so the separator position is
// deterministic. It returns ErrAPIKeyInvalid if the format is malformed.
func ParseAPIKey(fullKey string) (uuidKey, secret string, err error) {
	// Branch: reject keys without the expected scheme prefix.
	if !strings.HasPrefix(fullKey, "cl_") {
		err = NewCLError(ErrAPIKeyInvalid, "Invalid API key format: missing 'cl_' prefix", nil)
		return
	}
	rest := fullKey[3:]
	// Branch: a valid key is at least uuid(36) + separator(1) + hex-secret(64).
	if len(rest) < 101 {
		err = NewCLError(ErrAPIKeyInvalid, "Invalid API key format: too short", nil)
		return
	}
	// Branch: the 37th char (index 36) must be the uuid/secret separator.
	if rest[36] != '_' {
		err = NewCLError(ErrAPIKeyInvalid, "Invalid API key format: missing separator after UUID", nil)
		return
	}
	uuidKey = rest[:36]
	secret = rest[37:]
	// Branch: guard against an empty secret portion.
	if secret == "" {
		err = NewCLError(ErrAPIKeyInvalid, "Invalid API key format: empty secret", nil)
	}
	return
}

// isAPIKeyExpired reports whether the given expiry time is in the past.
// A nil expiry means the key never expires.
func isAPIKeyExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}
	return expiresAt.Before(time.Now())
}

// ValidateAPIKey authenticates a full plain-text key. It parses the key, looks
// it up by its UUID portion using a raw DB query (no membership scope, since the
// caller is not yet authenticated), and verifies it is not disabled, not expired,
// and that the secret matches the stored bcrypt hash.
// It returns the matching record, or one of ErrAPIKeyInvalid / ErrAPIKeyNotFound /
// ErrAPIKeyDisabled / ErrAPIKeyExpired describing why validation failed.
func (a *APIKeyAdmin) ValidateAPIKey(fullKey string) (apiKey *model.APIKey, err error) {
	// Note: never log fullKey or the secret — only the non-sensitive uuid below.
	uuidKey, secret, err := ParseAPIKey(fullKey)
	if err != nil {
		// Error handling: malformed key.
		logger.Errorf("Failed to parse API key: %v", err)
		return
	}
	logger.Debugf("Validating API key, uuid: %s", uuidKey)
	db := DB()
	apiKey = &model.APIKey{}
	if err = db.Where("api_key = ?", uuidKey).Take(apiKey).Error; err != nil {
		// Error handling: no key with this uuid.
		logger.Errorf("API key not found, uuid: %s: %v", uuidKey, err)
		err = NewCLError(ErrAPIKeyNotFound, "API key not found", err)
		return
	}
	// Branch: reject disabled keys.
	if apiKey.Disabled {
		logger.Errorf("API key is disabled, uuid: %s", uuidKey)
		err = NewCLError(ErrAPIKeyDisabled, "API key is disabled", nil)
		return
	}
	// Branch: reject expired keys.
	if isAPIKeyExpired(apiKey.ExpiresAt) {
		logger.Errorf("API key has expired, uuid: %s", uuidKey)
		err = NewCLError(ErrAPIKeyExpired, "API key has expired", nil)
		return
	}
	// Branch: the presented secret must match the stored bcrypt hash.
	if err = bcrypt.CompareHashAndPassword([]byte(apiKey.APIKeyHash), []byte(secret)); err != nil {
		logger.Errorf("API key secret mismatch, uuid: %s", uuidKey)
		err = NewCLError(ErrAPIKeyInvalid, "Invalid API key secret", err)
		return
	}
	logger.Debugf("Validated API key, uuid: %s", uuidKey)
	return
}
