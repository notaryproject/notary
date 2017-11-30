package chttp

import (
	"context"
	"net/url"
	"testing"
)

func TestDefaultAuth(t *testing.T) {
	dsn, err := url.Parse(dsn(t))
	if err != nil {
		t.Fatalf("Failed to parse DSN '%s': %s", dsn, err)
	}
	user := dsn.User.Username()
	client := getClient(t)

	if name := getAuthName(client, t); name != user {
		t.Errorf("Unexpected authentication name. Expected '%s', got '%s'", user, name)
	}

	if err = client.Logout(context.Background()); err != nil {
		t.Errorf("Failed to de-authenticate: %s", err)
	}

	if name := getAuthName(client, t); name != "" {
		t.Errorf("Unexpected authentication name after logout '%s'", name)
	}
}

func TestBasicAuth(t *testing.T) {
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
	ba := &BasicAuth{
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

func getAuthName(client *Client, t *testing.T) string {
	result := struct {
		Ctx struct {
			Name string `json:"name"`
		} `json:"userCtx"`
	}{}
	if _, err := client.DoJSON(context.Background(), "GET", "/_session", nil, &result); err != nil {
		t.Errorf("Failed to check session info: %s", err)
	}
	return result.Ctx.Name
}
