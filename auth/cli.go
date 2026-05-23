package auth

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func RunCLI(args []string) int {
	if len(args) == 0 {
		return runServe()
	}
	switch args[0] {
	case "serve":
		return runServe()
	case "login":
		return runLogin(args[1:])
	case "logout":
		return runLogout()
	case "status":
		return runStatus()
	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printUsage()
		return 2
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `DeltaDaemon MCP server

Usage:
  deltadaemon-mcp [serve]     Run MCP server (stdio)
  deltadaemon-mcp login       Sign in (email/password or --api-key)
  deltadaemon-mcp logout      Remove saved credentials
  deltadaemon-mcp status      Show login state

Credentials are stored in ~/.config/deltadaemon/credentials.json (mode 0600).
You do not need to put secrets in Cursor or Claude MCP config files.

Environment (optional):
  DELTADAEMON_API_BASE   Override API base URL
  DELTADAEMON_API_KEY    Override saved credentials (CI/scripts only)
`)
}

func runServe() int {
	// imported from main via callback to avoid circular imports
	return serveMCP()
}

var serveMCP func() int

func SetServeHandler(fn func() int) {
	serveMCP = fn
}

func runLogin(args []string) int {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	apiBase := fs.String("api-base", "", "API base URL (default https://api.deltadaemon.com/api/v1)")
	useAPIKey := fs.Bool("api-key", false, "paste an API key instead of email/password")
	_ = fs.Parse(args)

	opts := LoginOptions{APIBase: *apiBase}
	ctx := context.Background()

	if *useAPIKey {
		key := strings.TrimSpace(os.Getenv("DELTADAEMON_API_KEY"))
		if key == "" {
			var err error
			key, err = readSecret("API key: ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "read API key: %v\n", err)
				return 1
			}
		}
		opts.APIKey = key
	} else {
		email, err := readLine("Email: ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "read email: %v\n", err)
			return 1
		}
		password, err := readSecret("Password: ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "read password: %v\n", err)
			return 1
		}
		opts.Email = email
		opts.Password = password
	}

	if err := Login(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		return 1
	}
	path, _ := ConfigPath()
	fmt.Fprintf(os.Stderr, "Logged in. Credentials saved to %s\n", path)
	return 0
}

func runLogout() int {
	if err := Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "logout failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "Logged out.")
	return 0
}

func runStatus() int {
	st, err := CheckStatus(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if st != nil {
			fmt.Fprintf(os.Stderr, "Config: %s\n", st.ConfigPath)
		}
		return 1
	}
	fmt.Fprintf(os.Stderr, "Logged in (%s)\n", st.AuthMethod)
	if st.Email != "" {
		fmt.Fprintf(os.Stderr, "Email: %s\n", st.Email)
	}
	fmt.Fprintf(os.Stderr, "API base: %s\n", st.APIBase)
	fmt.Fprintf(os.Stderr, "Config: %s\n", st.ConfigPath)
	return 0
}

func readLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
