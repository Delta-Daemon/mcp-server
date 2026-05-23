package auth

import "testing"

func TestAuthBase(t *testing.T) {
	if got := AuthBase("https://api.deltadaemon.com/api/v1"); got != "https://api.deltadaemon.com" {
		t.Fatalf("AuthBase: got %q", got)
	}
	if got := AuthBase("http://localhost:8105/api/v1"); got != "http://localhost:8105" {
		t.Fatalf("AuthBase local: got %q", got)
	}
}

func TestTrimBase(t *testing.T) {
	if got := trimBase("https://api.deltadaemon.com/api/v1/"); got != "https://api.deltadaemon.com/api/v1" {
		t.Fatalf("trimBase: got %q", got)
	}
}
