package auth

import (
	"strings"
	"testing"
)

func TestMCPLoginURL(t *testing.T) {
	got, err := mcpLoginURL("https://api.deltadaemon.com/api/v1", "google", "http://127.0.0.1:8765/callback", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "https://api.deltadaemon.com/auth/mcp/login") {
		t.Fatalf("unexpected url: %s", got)
	}
	if !strings.Contains(got, "provider=google") {
		t.Fatalf("missing provider: %s", got)
	}
	if !strings.Contains(got, "state=abc123") {
		t.Fatalf("missing state: %s", got)
	}
}

func TestRandomToken(t *testing.T) {
	a, err := randomToken(16)
	if err != nil {
		t.Fatal(err)
	}
	b, err := randomToken(16)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || len(a) != 32 {
		t.Fatalf("unexpected token: %q", a)
	}
}
