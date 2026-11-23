package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"

	"gorm.io/gorm"
)

// Scope constants for the machine API. node-sync lets a remote master read
// this panel's status and traffic; node-admin additionally allows the master
// to manage inbounds and clients on this panel.
const (
	ScopeNodeSync  = "node-sync"
	ScopeNodeAdmin = "node-admin"
)

var (
	ErrTokenNotFound = errors.New("api token not found")
	ErrTokenNoScope  = errors.New("api token lacks the required scope")
)

// ApiTokenService manages bearer tokens for the machine-to-machine API
// (multi-panel Nodes). The plaintext is returned exactly once at creation;
// only a SHA-256 hash is persisted.
type ApiTokenService struct{}

// Create generates a new token with the given scopes (comma-separated) and
// returns it together with the stored model. The plaintext never leaves this
// return value.
func (s *ApiTokenService) Create(name string, scopes string) (string, *model.ApiToken, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil, errors.New("token name is required")
	}
	if scopes == "" {
		scopes = ScopeNodeSync
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	plaintext := "vpnaio_" + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plaintext))
	token := &model.ApiToken{
		Name:      name,
		TokenHash: hex.EncodeToString(sum[:]),
		Scopes:    scopes,
	}
	if err := database.GetDB().Create(token).Error; err != nil {
		return "", nil, err
	}
	return plaintext, token, nil
}

// Verify resolves a plaintext bearer token to the stored model and records its
// last use. ErrTokenNotFound when the hash is unknown.
func (s *ApiTokenService) Verify(plaintext string) (*model.ApiToken, error) {
	if plaintext == "" {
		return nil, ErrTokenNotFound
	}
	sum := sha256.Sum256([]byte(plaintext))
	hash := hex.EncodeToString(sum[:])
	token := &model.ApiToken{}
	if err := database.GetDB().Where("token_hash = ?", hash).First(token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	database.GetDB().Model(token).Update("last_used_at", time.Now().Unix())
	return token, nil
}

// HasScope reports whether the token carries the given scope.
func (s *ApiTokenService) HasScope(token *model.ApiToken, scope string) bool {
	for _, s := range strings.Split(token.Scopes, ",") {
		if strings.TrimSpace(s) == scope {
			return true
		}
	}
	return false
}

// List returns every token ordered by creation.
func (s *ApiTokenService) List() ([]*model.ApiToken, error) {
	tokens := make([]*model.ApiToken, 0)
	err := database.GetDB().Order("id asc").Find(&tokens).Error
	return tokens, err
}

// Revoke deletes a token by id.
func (s *ApiTokenService) Revoke(id int) error {
	result := database.GetDB().Delete(&model.ApiToken{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: id %d", ErrTokenNotFound, id)
	}
	return nil
}

// EnsureTokenHashMatches is a constant-time comparison helper kept for tests
// and future challenge flows; it never short-circuits on the first byte.
func EnsureTokenHashMatches(hash string, plaintext string) bool {
	sum := sha256.Sum256([]byte(plaintext))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(hash), []byte(expected)) == 1
}
