package resources

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Delta-Daemon/mcp-server/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed openapi.yaml
var openAPISpec []byte

const overviewURI = "deltadaemon://docs/overview"
const openAPIURI = "deltadaemon://docs/openapi"

func Register(server *mcp.Server, api *client.Client) {
	server.AddResource(&mcp.Resource{
		URI:         overviewURI,
		Name:        "DeltaDaemon API overview",
		Description: "Authentication, conventions, and endpoint summary for building apps against the DeltaDaemon forecast-accuracy API.",
		MIMEType:    "text/markdown",
	}, readOverview)

	server.AddResource(&mcp.Resource{
		URI:         openAPIURI,
		Name:        "DeltaDaemon OpenAPI spec",
		Description: "Full OpenAPI 3 spec for code generation and endpoint discovery.",
		MIMEType:    "application/yaml",
	}, readOpenAPI)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "deltadaemon://stations/{station_id}",
		Name:        "Station metadata",
		Description: "Metadata for one ICAO weather station (coordinates, timezone, climate zone).",
		MIMEType:    "application/json",
	}, readStation(api))
}

func readOverview(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	text := fmt.Sprintf(`# DeltaDaemon API

Base URL: %s

## Authentication
- **Public:** `+"`/stations/metadata`"+`, `+"`/public/*`"+` — no auth (public routes are rate-limited by IP).
- **Data & accuracy:** `+"`Authorization: Bearer <API_KEY>`"+` — get a key at https://deltadaemon.com/signin

## Conventions
- Temperatures in **Fahrenheit** unless `+"`temp_unit=celsius`"+`.
- `+"`metric=high`"+` (default): daily max forecast vs observed high. `+"`metric=low`"+`: daily min.
- **Error** = forecast − observed. Positive = forecast too warm.
- **Date window:** `+"`days`"+` (rolling, default 90) or `+"`date_from`"+` / `+"`date_to`"+` (YYYY-MM-DD).

## Response envelope
`+"```json"+`
{"success": true, "data": {...}, "metadata": {"generated_at": "...", ...}}
`+"```"+`

## Key accuracy fields
| Field | Meaning |
|-------|---------|
| count | sample size |
| mean_error | bias (°F) |
| mae | mean absolute error |
| rmse | root mean square error |
| std_dev | error spread |

## MCP tools
Use `+"`list_stations`"+` first, then accuracy tools (`+"`get_accuracy_summary`"+`, `+"`get_station_accuracy`"+`, `+"`get_exceedance`"+`, etc.).
For unlisted GET routes, use `+"`query_deltadaemon_api`"+`.

Full spec: resource `+"`%s`"+`.`, "https://api.deltadaemon.com/api/v1", openAPIURI)

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: overviewURI, MIMEType: "text/markdown", Text: text},
		},
	}, nil
}

func readOpenAPI(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: openAPIURI, MIMEType: "application/yaml", Text: string(openAPISpec)},
		},
	}, nil
}

func readStation(api *client.Client) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		stationID, err := parseStationURI(req.Params.URI)
		if err != nil {
			return nil, err
		}
		raw, err := api.Get(ctx, "/stations/metadata", nil)
		if err != nil {
			return nil, err
		}
		var envelope struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, err
		}
		want := strings.ToUpper(stationID)
		for _, st := range envelope.Data {
			if id, _ := st["station_id"].(string); strings.EqualFold(id, want) {
				out, err := json.MarshalIndent(st, "", "  ")
				if err != nil {
					return nil, err
				}
				return &mcp.ReadResourceResult{
					Contents: []*mcp.ResourceContents{
						{URI: req.Params.URI, MIMEType: "application/json", Text: string(out)},
					},
				}, nil
			}
		}
		return nil, fmt.Errorf("station not found: %s", stationID)
	}
}

func parseStationURI(uri string) (string, error) {
	const prefix = "deltadaemon://stations/"
	if len(uri) <= len(prefix) || uri[:len(prefix)] != prefix {
		return "", fmt.Errorf("invalid station URI: %s", uri)
	}
	id := uri[len(prefix):]
	if id == "" {
		return "", fmt.Errorf("station_id missing in URI")
	}
	return id, nil
}
