package authdb

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestCreateAuthToken(t *testing.T) {
	type catTest struct {
		name                   string
		username, salt, secret string
		time                   int64
		expected               string
		recovery               string
	}
	tests := []catTest{
		{
			name:     "no secret",
			recovery: "secret must be set",
		},
		{
			name:     "no salt",
			secret:   "foo",
			recovery: "salt must be set",
		},
		{
			name:     "valid",
			secret:   "foo",
			salt:     "4e170ffeb6f34daecfd814dfb4001a73",
			username: "baz",
			time:     12345,
			expected: "YmF6OjMwMzk6fqvvtjD0eY7M_OZCiSWeRk01mlo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recovery := func() (recovery string) {
				defer func() {
					if r := recover(); r != nil {
						recovery = r.(string)
					}
				}()
				result := CreateAuthToken(test.username, test.salt, test.secret, test.time)
				if result != test.expected {
					t.Errorf("Unexpected result: %s", result)
				}
				return ""
			}()
			if recovery != test.recovery {
				t.Errorf("Unexpected panic recovery: %s", recovery)
			}
		})
	}
}

func TestDecodeAuthToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		username string
		created  time.Time
		err      string
	}{
		{
			name:  "invalid base64",
			input: "ée",
			err:   "illegal base64 data at input byte 0",
		},
		{
			name:  "invalid payload",
			input: base64.RawURLEncoding.EncodeToString([]byte("foo bar baz")),
			err:   "invalid payload",
		},
		{
			name:  "invalid timestamp",
			input: base64.RawURLEncoding.EncodeToString([]byte("foo:asdf:asdf")),
			err:   "invalid timestamp 'asdf'",
		},
		{
			name:     "valid",
			input:    base64.RawURLEncoding.EncodeToString([]byte("foo:12345:asdf")),
			username: "foo",
			created:  time.Unix(74565, 0),
		},
		{
			name:     "real world token",
			input:    "MzBkMzRmODktMGUyMC00YzZhLTgyZjQtN2FhOWEyMmZkYThmOjU5OTlBNDI0OlGqCaGA69un9MHg2_Cyd95h4zkH",
			username: "30d34f89-0e20-4c6a-82f4-7aa9a22fda8f",
			created:  time.Unix(1503241252, 0),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			username, created, err := DecodeAuthToken(test.input)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != test.err {
				t.Errorf("Unexpected error: %s", errMsg)
			}
			if err != nil {
				return
			}
			if test.username != username || !test.created.Equal(created) {
				t.Errorf("Unexpected results: %s / %v\n", username, created)
			}
		})
	}
}
