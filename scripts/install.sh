#!/usr/bin/env sh
# Install deltadaemon-mcp to ~/.local/bin (or $INSTALL_DIR).
set -eu

REPO="${DELTADAEMON_MCP_REPO:-github.com/Delta-Daemon/mcp-server}"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
BINARY="deltadaemon-mcp"

mkdir -p "$INSTALL_DIR"

install_with_go() {
	if ! command -v go >/dev/null 2>&1; then
		return 1
	fi
	echo "Installing with go install $REPO@latest ..."
	GOBIN="$INSTALL_DIR" go install "${REPO}@latest"
}

install_from_release() {
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	case "$arch" in
		x86_64|amd64) arch="amd64" ;;
		arm64|aarch64) arch="arm64" ;;
		*) return 1 ;;
	esac
	case "$os" in
		darwin|linux) ;;
		*) return 1 ;;
	esac

	url="https://github.com/Delta-Daemon/mcp-server/releases/latest/download/${BINARY}_${os}_${arch}.tar.gz"
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT INT TERM

	if ! curl -fsSL "$url" -o "$tmp/release.tar.gz" 2>/dev/null; then
		return 1
	fi
	tar -xzf "$tmp/release.tar.gz" -C "$tmp"
	install -m 0755 "$tmp/$BINARY" "$INSTALL_DIR/$BINARY"
	echo "Installed from GitHub release."
	return 0
}

if install_from_release; then
	:
elif install_with_go; then
	:
else
	echo "Could not install from release or go install." >&2
	echo "Install Go from https://go.dev/dl/ and re-run, or build from source:" >&2
	echo "  git clone git@github.com:Delta-Daemon/mcp-server.git" >&2
	echo "  cd mcp-server && go build -o $INSTALL_DIR/$BINARY ." >&2
	exit 1
fi

if ! echo ":$PATH:" | grep -q ":${INSTALL_DIR}:"; then
	echo ""
	echo "Add $INSTALL_DIR to your PATH, then run:"
	echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo ""
echo "Next steps:"
echo "  deltadaemon-mcp setup"
