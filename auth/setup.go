package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func RunSetup(args []string) int {
	ctx := context.Background()
	binary, err := executablePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		return 1
	}

	if _, err := CheckStatus(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Sign in to connect DeltaDaemon to your editor.")
		if runLogin([]string{}) != 0 {
			return 1
		}
	} else {
		fmt.Fprintln(os.Stderr, "Already signed in.")
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Add this to your MCP config:")
	fmt.Fprintln(os.Stderr)
	printMCPConfig(os.Stdout, binary)
	fmt.Fprintln(os.Stderr)

	printConfigPaths(os.Stderr)
	return 0
}

func executablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

func printMCPConfig(w interface{ Write([]byte) (int, error) }, command string) {
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"deltadaemon": map[string]any{
				"command": command,
			},
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(cfg)
}

func printConfigPaths(w interface{ Write([]byte) (int, error) }) {
	fmt.Fprintln(w, "Config file locations:")
	if runtime.GOOS == "darwin" {
		fmt.Fprintln(w, "  Cursor:        ~/.cursor/mcp.json")
		fmt.Fprintln(w, "  Claude Desktop: ~/Library/Application Support/Claude/claude_desktop_config.json")
		return
	}
	if runtime.GOOS == "windows" {
		fmt.Fprintln(w, "  Cursor:         USERPROFILE\\.cursor\\mcp.json")
		fmt.Fprintln(w, "  Claude Desktop: APPDATA\\Claude\\claude_desktop_config.json")
		return
	}
	fmt.Fprintln(w, "  Cursor:        ~/.cursor/mcp.json")
	fmt.Fprintln(w, "  Claude Desktop: ~/.config/Claude/claude_desktop_config.json")
}

func InstallDir() string {
	if d := strings.TrimSpace(os.Getenv("INSTALL_DIR")); d != "" {
		return d
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Programs", "deltadaemon")
		}
	}
	if d := os.Getenv("XDG_BIN_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "bin")
}

func OnPath(dir string) bool {
	if dir == "" {
		return false
	}
	for _, part := range filepath.SplitList(os.Getenv("PATH")) {
		if part == dir {
			return true
		}
	}
	return false
}

func SuggestPathFix(dir string) string {
	if dir == "" || OnPath(dir) {
		return ""
	}
	shell := filepath.Base(os.Getenv("SHELL"))
	if shell == "fish" {
		return fmt.Sprintf("Add to PATH: fish -c 'fish_add_path %s'", dir)
	}
	if shell == "zsh" {
		return fmt.Sprintf("Add to PATH: echo 'export PATH=%q:$PATH' >> ~/.zshrc", dir)
	}
	return fmt.Sprintf("Add to PATH: export PATH=%q:$PATH", dir)
}

func HasGo() bool {
	_, err := exec.LookPath("go")
	return err == nil
}
