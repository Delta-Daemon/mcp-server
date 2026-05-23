# DeltaDaemon MCP Server

Go [Model Context Protocol](https://modelcontextprotocol.io) server that exposes DeltaDaemon forecast-accuracy data to **Cursor**, **Claude Desktop**, and other MCP clients.

## Quick start

```bash
curl -fsSL https://raw.githubusercontent.com/Delta-Daemon/mcp-server/main/scripts/install.sh | sh
deltadaemon-mcp setup
```

`setup` opens your browser for Google sign-in (or GitHub with `--provider github`), saves credentials locally, and prints the MCP config snippet for your editor.

No secrets go in your MCP config — only the binary path.

## Install options

**Install script** (recommended):

```bash
curl -fsSL https://raw.githubusercontent.com/Delta-Daemon/mcp-server/main/scripts/install.sh | sh
```

The script tries a GitHub release binary first, then `go install`.

**Go install:**

```bash
go install github.com/Delta-Daemon/mcp-server@latest
```

**Build from source:**

```bash
git clone git@github.com:Delta-Daemon/mcp-server.git
cd mcp-server
go build -o deltadaemon-mcp .
./deltadaemon-mcp setup
```

## Cursor / Claude MCP config

After `setup`, paste the printed JSON into your MCP config:

| Editor | Config file |
|--------|-------------|
| Cursor | `~/.cursor/mcp.json` |
| Claude Desktop (macOS) | `~/Library/Application Support/Claude/claude_desktop_config.json` |

Example:

```json
{
  "mcpServers": {
    "deltadaemon": {
      "command": "/Users/you/.local/bin/deltadaemon-mcp"
    }
  }
}
```

Restart your editor after saving the config.

## Authentication

| Method | Command |
|--------|---------|
| **Browser OAuth** (default) | `deltadaemon-mcp login` |
| **GitHub OAuth** | `deltadaemon-mcp login --provider github` |
| **API key** | `deltadaemon-mcp login --api-key` |
| **Email/password** | `deltadaemon-mcp login --password` |

Credentials are saved to `~/.config/deltadaemon/credentials.json` (mode `0600`).

Check login state:

```bash
deltadaemon-mcp status
deltadaemon-mcp logout
```

Priority: `DELTADAEMON_API_KEY` env → saved API key → saved session cookie.

## Optional environment

| Variable | Default | Description |
|----------|---------|-------------|
| `DELTADAEMON_API_BASE` | `https://api.deltadaemon.com/api/v1` | API base (local: `http://localhost:8105/api/v1`) |
| `DELTADAEMON_API_KEY` | — | Skip saved credentials (automation only) |

## Commands

| Command | Description |
|---------|-------------|
| `deltadaemon-mcp` or `serve` | Run MCP server over stdio |
| `setup` | Sign in + print MCP config |
| `login` | Sign in with Google/GitHub OAuth |
| `logout` | Delete saved credentials |
| `status` | Verify login and show config path |

## Tools

| Tool | Description |
|------|-------------|
| `list_stations` | All tracked stations (public, no login) |
| `get_accuracy_summary` | MAE, bias, RMSE for a station/city |
| `get_accuracy_by_city` | City rankings |
| `get_station_accuracy` | Per-station detail (+ optional raw samples) |
| `get_accuracy_by_lead_time` | Lead-time buckets |
| `get_accuracy_by_weather_regime` | Clear/cloudy/precip regimes |
| `get_exceedance` | Error threshold exceedance rates |
| `get_hit_rate` | Exact / ±1°F / ±2°F hit rates |
| `get_bias_correction` | Bias-adjust a forecast |
| `get_error_distribution` | Error histogram |
| `get_forecast_actual_pairs` | Raw pairs |
| `get_hourly_snapshot` | Single NWS run snapshot |
| `query_deltadaemon_api` | Generic GET to any `/api/v1` path |

Common query params: `station_id`, `city`, `days`, `date_from`, `date_to`, `metric` (`high` or `low`).

## Resources

| URI | Content |
|-----|---------|
| `deltadaemon://docs/overview` | API conventions (markdown) |
| `deltadaemon://docs/openapi` | Embedded OpenAPI spec |
| `deltadaemon://stations/{station_id}` | One station's metadata |

## Prompts

| Prompt | Use |
|--------|-----|
| `interpret_accuracy` | Explain metrics for trading/risk |
| `build_weather_app` | Scaffold an app against the API |

## Local API

```bash
deltadaemon-mcp login --api-base http://localhost:8105/api/v1
deltadaemon-mcp serve
```

The API must expose `GET /auth/mcp/login` for browser OAuth (included in delta-daemon-api).

> **Note:** Browser OAuth requires the Deltadaemon API to be deployed with the MCP auth changes (`handlers/auth_mcp.go`, updated `handlers/auth_oauth.go`, route in `main.go`). Until that's deployed to production, use `--api-key` or `--password` as fallbacks.
