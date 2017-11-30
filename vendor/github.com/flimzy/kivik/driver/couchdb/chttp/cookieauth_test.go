// +build !js
// This test doesn't work in nodejs

package chttp

import (
	"context"
	"net/url"
	"testing"
)

func TestCookieAuth(t *testing.T) {
	dsn, err := url.Parse(dsn(t))
	if err != nil {
		t.Fatalf("Failed to parse DSN '%s': %s", dsn, err)
	}
	user := dsn.User
	dsn.User = nil
	client, err := New(context.Background(), dsn.String())
	if err != nil {
		t.Fatalf("Failed to connect: %s", err)
	}
	if name := getAuthName(client, t); name != "" {
		t.Errorf("Unexpected authentication name '%s'", name)
	}

	if err = client.Logout(context.Background()); err == nil {
		t.Errorf("Logout should have failed prior to login")
	}

	password, _ := user.Password()
	ba := &CookieAuth{
		Username: user.Username(),
		Password: password,
	}
	if err = client.Auth(context.Background(), ba); err != nil {
		t.Errorf("Failed to authenticate: %s", err)
	}
	if err = client.Auth(context.Background(), ba); err == nil {
		t.Errorf("Expected error trying to double-auth")
	}
	if name := getAuthName(client, t); name != user.Username() {
		t.Errorf("Unexpected auth name. Expected '%s', got '%s'", user.Username(), name)
	}

	if err = client.Logout(context.Background()); err != nil {
		t.Errorf("Failed to de-authenticate: %s", err)
	}

	if name := getAuthName(client, t); name != "" {
		t.Errorf("Unexpected authentication name after logout '%s'", name)
	}
}
