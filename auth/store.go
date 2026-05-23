package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Credentials struct {
	APIBase string `json:"api_base,omitempty"`
	Email   string `json:"email,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	Session string `json:"session,omitempty"`
}

func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func configDir() (string, error) {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "deltadaemon"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "deltadaemon"), nil
}

func Load() (*Credentials, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cred Credentials
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	return &cred, nil
}

func Save(cred *Credentials) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "credentials.json")
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return nil
}

func Clear() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func Resolve() (apiBase, apiKey, session string) {
	apiBase = defaultAPIBase()
	if v := os.Getenv("DELTADAEMON_API_BASE"); v != "" {
		apiBase = trimBase(v)
	}
	if v := os.Getenv("DELTADAEMON_API_KEY"); v != "" {
		return apiBase, v, ""
	}
	cred, err := Load()
	if err != nil || cred == nil {
		return apiBase, "", ""
	}
	if cred.APIBase != "" {
		apiBase = trimBase(cred.APIBase)
	}
	if cred.APIKey != "" {
		return apiBase, cred.APIKey, ""
	}
	return apiBase, "", cred.Session
}

func AuthBase(apiBase string) string {
	base := trimBase(apiBase)
	if strings.HasSuffix(base, "/api/v1") {
		return strings.TrimSuffix(base, "/api/v1")
	}
	return base
}

func defaultAPIBase() string {
	return "https://api.deltadaemon.com/api/v1"
}

func trimBase(s string) string {
	return strings.TrimRight(s, "/")
}
