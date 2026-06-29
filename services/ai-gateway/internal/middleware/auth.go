package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

type Authenticator struct {
	hashes [][]byte
}

func NewAuthenticator(hexHashes []string) (*Authenticator, error) {
	auth := &Authenticator{hashes: make([][]byte, 0, len(hexHashes))}
	for _, raw := range hexHashes {
		decoded, err := hex.DecodeString(strings.TrimSpace(raw))
		if err != nil || len(decoded) != sha256.Size {
			return nil, errInvalidTokenHash
		}
		auth.hashes = append(auth.hashes, decoded)
	}
	return auth, nil
}

func (a *Authenticator) Verify(token string) bool {
	if a == nil || len(a.hashes) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return false
	}
	sum := sha256.Sum256([]byte(trimmed))
	for _, hash := range a.hashes {
		if subtle.ConstantTimeCompare(sum[:], hash) == 1 {
			return true
		}
	}
	return false
}

type invalidTokenHashError struct{}

func (invalidTokenHashError) Error() string { return "invalid service token hash" }

var errInvalidTokenHash invalidTokenHashError
