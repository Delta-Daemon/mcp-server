package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type LoginOptions struct {
	APIBase  string
	Email    string
	Password string
	APIKey   string
}

type AccountStatus struct {
	APIBase    string
	Email      string
	AuthMethod string
	ConfigPath string
}

func Login(ctx context.Context, opts LoginOptions) error {
	apiBase := trimBase(opts.APIBase)
	if apiBase == "" {
		apiBase = defaultAPIBase()
	}

	cred := &Credentials{APIBase: apiBase}

	if opts.APIKey != "" {
		cred.APIKey = strings.TrimSpace(opts.APIKey)
		if err := verifyAPIKey(ctx, apiBase, cred.APIKey); err != nil {
			return err
		}
	} else {
		if opts.Email == "" || opts.Password == "" {
			return fmt.Errorf("email and password are required (or use --api-key)")
		}
		session, email, err := loginWithPassword(ctx, apiBase, opts.Email, opts.Password)
		if err != nil {
			return err
		}
		cred.Session = session
		cred.Email = email
	}

	return Save(cred)
}

func CheckStatus(ctx context.Context) (*AccountStatus, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	apiBase, apiKey, session := Resolve()
	st := &AccountStatus{APIBase: apiBase, ConfigPath: path}

	if apiKey != "" {
		if os.Getenv("DELTADAEMON_API_KEY") != "" {
			st.AuthMethod = "api_key (environment)"
		} else {
			st.AuthMethod = "api_key (saved)"
		}
		if err := verifyAPIKey(ctx, apiBase, apiKey); err != nil {
			return st, fmt.Errorf("saved API key is invalid: %w", err)
		}
		return st, nil
	}
	if session != "" {
		email, err := verifySession(ctx, apiBase, session)
		if err != nil {
			return st, fmt.Errorf("saved session expired; run: deltadaemon-mcp login")
		}
		st.Email = email
		if os.Getenv("DELTADAEMON_API_KEY") == "" {
			st.AuthMethod = "session (saved)"
		}
		return st, nil
	}
	return st, fmt.Errorf("not logged in; run: deltadaemon-mcp login")
}

func loginWithPassword(ctx context.Context, apiBase, email, password string) (sessionID, userEmail string, err error) {
	body, _ := json.Marshal(map[string]string{
		"email":    strings.TrimSpace(strings.ToLower(email)),
		"password": password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AuthBase(apiBase)+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("login failed: %s", truncate(string(respBody), 256))
	}

	for _, c := range resp.Cookies() {
		if c.Name == "dd_session" && c.Value != "" {
			sessionID = c.Value
			break
		}
	}
	if sessionID == "" {
		return "", "", fmt.Errorf("login succeeded but no session cookie returned")
	}

	userEmail, err = verifySession(ctx, apiBase, sessionID)
	if err != nil {
		return "", "", err
	}
	return sessionID, userEmail, nil
}

func verifySession(ctx context.Context, apiBase, sessionID string) (email string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AuthBase(apiBase)+"/auth/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.AddCookie(&http.Cookie{Name: "dd_session", Value: sessionID})

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("session check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("session invalid or expired")
	}

	var out struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("parse session response: %w", err)
	}
	return out.User.Email, nil
}

func verifyAPIKey(ctx context.Context, apiBase, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/accuracy/freshness", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("API key check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusPaymentRequired {
		return fmt.Errorf("API key invalid or plan inactive")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("API key check: %s", truncate(string(body), 256))
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
