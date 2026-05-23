# DeltaDaemon MCP Server

Go [Model Context Protocol](https://modelcontextprotocol.io) server that exposes DeltaDaemon forecast-accuracy data to **Cursor**, **Claude Desktop**, and other MCP clients.

## Setup (one time)

Sign in once from a terminal. Credentials are saved to `~/.config/deltadaemon/credentials.json` (file mode `0600`) — **not** in your MCP config.

```bash
git clone git@github.com:Delta-Daemon/mcp-server.git
cd mcp-server
go build -o deltadaemon-mcp .
./deltadaemon-mcp login
```

Use your [DeltaDaemon account](https://deltadaemon.com/signin) email and password. Prefer an API key instead?

```bash
./deltadaemon-mcp login --api-key
# or non-interactive:
DELTADAEMON_API_KEY=dd_live_... ./deltadaemon-mcp login --api-key
```

Check login state:

```bash
./deltadaemon-mcp status
./deltadaemon-mcp logout   # remove saved credentials
```

## Cursor / Claude MCP config

No secrets in the config — only the command path:

```json
{
  "mcpServers": {
    "deltadaemon": {
      "command": "/absolute/path/to/mcp-server/deltadaemon-mcp"
    }
  }
}
```

Run `./deltadaemon-mcp login` before first use. The MCP server reads saved credentials automatically.

## How auth works

| Method | When |
|--------|------|
| **Saved session** (default) | `deltadaemon-mcp login` with email/password |
| **Saved API key** | `deltadaemon-mcp login --api-key` |
| **Environment override** | `DELTADAEMON_API_KEY` for CI/scripts only |

Priority: `DELTADAEMON_API_KEY` env → saved API key → saved session cookie.

The DeltaDaemon API accepts either a Bearer API key or a browser-style `dd_session` cookie; the MCP server uses whichever you saved.

## Optional environment

| Variable | Default | Description |
|----------|---------|-------------|
| `DELTADAEMON_API_BASE` | `https://api.deltadaemon.com/api/v1` | API base (local: `http://localhost:8105/api/v1`) |
| `DELTADAEMON_API_KEY` | — | Skip saved credentials (automation only) |

## Commands

| Command | Description |
|---------|-------------|
| `deltadaemon-mcp` or `serve` | Run MCP server over stdio |
| `login` | Save credentials interactively |
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
./deltadaemon-mcp login --api-base http://localhost:8105/api/v1
./deltadaemon-mcp serve
```
