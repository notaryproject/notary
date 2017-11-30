// Package authdb provides a standard interface to an authentication user store
// to be used by AuthHandlers.
package authdb

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// A UserStore provides an AuthHandler with access to a user store for.
type UserStore interface {
	// Validate returns a user context object if the credentials are valid. An
	// error must be returned otherwise. A Not Found error must not be returned.
	// Not Found should be treated identically to Unauthorized.
	Validate(ctx context.Context, username, password string) (user *UserContext, err error)
	// UserCtx returns a user context object if the user exists. It is used by
	// AuthHandlers that don't validate the password (e.g. Cookie auth).
	UserCtx(ctx context.Context, username string) (user *UserContext, err error)
}

// PBKDF2KeyLength is the key length, in bytes, of the PBKDF2 keys used by
// CouchDB.
const PBKDF2KeyLength = 20

// SchemePBKDF2 is the default CouchDB password scheme.
const SchemePBKDF2 = "pbkdf2"

// UserContext represents a CouchDB UserContext object.
// See http://docs.couchdb.org/en/2.0.0/json-structure.html#userctx-object.
type UserContext struct {
	Database string   `json:"db,omitempty"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	// Salt is needed to calculate cookie tokens.
	Salt string `json:"-"`
}

// ValidatePBKDF2 returns true if the calculated hash matches the derivedKey.
func ValidatePBKDF2(password, salt, derivedKey string, iterations int) bool {
	hash := fmt.Sprintf("%x", pbkdf2.Key([]byte(password), []byte(salt), iterations, PBKDF2KeyLength, sha1.New))
	return hash == derivedKey
}

// CreateAuthToken hashes a username, salt, timestamp, and the server secret
// into an authentication token.
func CreateAuthToken(name, salt, secret string, time int64) string {
	if secret == "" {
		panic("secret must be set")
	}
	if salt == "" {
		panic("salt must be set")
	}
	sessionData := fmt.Sprintf("%s:%X", name, time)
	h := hmac.New(sha1.New, []byte(secret+salt))
	_, _ = h.Write([]byte(sessionData))
	hashData := string(h.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(sessionData + ":" + hashData))
}

// MarshalJSON satisfies the json.Marshaler interface.
func (c *UserContext) MarshalJSON() ([]byte, error) {
	roles := c.Roles
	if roles == nil {
		roles = []string{}
	}
	output := map[string]interface{}{
		"roles": roles,
	}
	if c.Database != "" {
		output["db"] = c.Database
	}
	if c.Name != "" {
		output["name"] = c.Name
	} else {
		output["name"] = nil
	}
	return json.Marshal(output)
}

// DecodeAuthToken decodes an auth token, extracting the username and token
// token creation time. To validate the authenticity of the token, use
// ValidatePBKDF2().
func DecodeAuthToken(token string) (username string, created time.Time, err error) {
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return username, created, err
	}
	parts := bytes.SplitN(payload, []byte(":"), 3)
	if len(parts) < 3 {
		return username, created, errors.New("invalid payload")
	}
	seconds, err := strconv.ParseInt(string(parts[1]), 16, 64)
	if err != nil {
		return username, created, fmt.Errorf("invalid timestamp '%s'", string(parts[1]))
	}
	return string(parts[0]), time.Unix(seconds, 0), nil
}
