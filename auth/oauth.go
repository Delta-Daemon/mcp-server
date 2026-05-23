package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type BrowserLoginOptions struct {
	APIBase   string
	Provider  string
	Timeout   time.Duration
	NoBrowser bool
}

type browserCallback struct {
	Session string
	Email   string
	State   string
	Err     error
}

func LoginWithBrowser(ctx context.Context, opts BrowserLoginOptions) error {
	apiBase := trimBase(opts.APIBase)
	if apiBase == "" {
		apiBase = defaultAPIBase()
	}

	provider := strings.TrimSpace(strings.ToLower(opts.Provider))
	if provider == "" {
		provider = "google"
	}
	if provider != "google" && provider != "github" {
		return fmt.Errorf("provider must be google or github")
	}

	state, err := randomToken(16)
	if err != nil {
		return err
	}

	listener, port, err := listenLoopback()
	if err != nil {
		return err
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	resultCh := make(chan browserCallback, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			writeCallbackPage(w, false, "Sign-in failed: state mismatch. Close this tab and run deltadaemon-mcp login again.")
			resultCh <- browserCallback{Err: fmt.Errorf("oauth state mismatch")}
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			writeCallbackPage(w, false, "Sign-in failed: "+errMsg)
			resultCh <- browserCallback{Err: fmt.Errorf("oauth error: %s", errMsg)}
			return
		}
		session := strings.TrimSpace(r.URL.Query().Get("session"))
		if session == "" {
			writeCallbackPage(w, false, "Sign-in failed: no session returned.")
			resultCh <- browserCallback{Err: fmt.Errorf("oauth callback missing session")}
			return
		}

		email, err := verifySession(r.Context(), apiBase, session)
		if err != nil {
			writeCallbackPage(w, false, "Sign-in failed: could not verify session.")
			resultCh <- browserCallback{Err: err}
			return
		}

		writeCallbackPage(w, true, "Signed in to DeltaDaemon. You can close this tab and return to your terminal.")
		resultCh <- browserCallback{Session: session, Email: email, State: state}
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	loginURL, err := mcpLoginURL(apiBase, provider, redirectURI, state)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Opening browser for %s sign-in…\n", provider)
	fmt.Fprintf(os.Stderr, "If the browser does not open, visit:\n%s\n\n", loginURL)
	if !opts.NoBrowser {
		_ = openBrowser(loginURL)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-resultCh:
		if res.Err != nil {
			return res.Err
		}
		return Save(&Credentials{
			APIBase: apiBase,
			Email:   res.Email,
			Session: res.Session,
		})
	case <-time.After(timeout):
		return fmt.Errorf("sign-in timed out after %s — try again", timeout)
	}
}

func mcpLoginURL(apiBase, provider, redirectURI, state string) (string, error) {
	u, err := url.Parse(AuthBase(apiBase) + "/auth/mcp/login")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("provider", provider)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func listenLoopback() (net.Listener, int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, err
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		listener.Close()
		return nil, 0, fmt.Errorf("unexpected listener address type")
	}
	return listener, addr.Port, nil
}

func randomToken(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(target string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Run()
	default:
		if err := exec.Command("xdg-open", target).Run(); err == nil {
			return nil
		}
		for _, bin := range []string{"sensible-browser", "x-www-browser", "www-browser"} {
			if err := exec.Command(bin, target).Run(); err == nil {
				return nil
			}
		}
		return fmt.Errorf("could not open browser")
	}
}

func writeCallbackPage(w http.ResponseWriter, ok bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := "DeltaDaemon MCP"
	if ok {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		title = "Sign-in failed"
	}
	_ = callbackPage.Execute(w, map[string]string{
		"Title":   title,
		"Message": message,
		"OK":      fmt.Sprintf("%t", ok),
	})
}

var callbackPage = template.Must(template.New("callback").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{.Title}}</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 32rem; margin: 4rem auto; padding: 0 1rem; color: #111; }
    .ok { color: #0a7; }
    .err { color: #c33; }
  </style>
</head>
<body>
  <h1 class="{{if eq .OK "true"}}ok{{else}}err{{end}}">{{.Title}}</h1>
  <p>{{.Message}}</p>
</body>
</html>`))

