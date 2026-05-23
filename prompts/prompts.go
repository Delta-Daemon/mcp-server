package prompts

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Register(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "interpret_accuracy",
		Description: "Explain DeltaDaemon accuracy metrics and how to use them for trading or risk decisions.",
		Arguments: []*mcp.PromptArgument{
			{Name: "station_id", Description: "ICAO station (e.g. KLAX)", Required: false},
			{Name: "use_case", Description: "e.g. prediction market, hedging, dashboard", Required: false},
		},
	}, interpretAccuracy)

	server.AddPrompt(&mcp.Prompt{
		Name:        "build_weather_app",
		Description: "Scaffold an app that consumes DeltaDaemon forecast-accuracy data.",
		Arguments: []*mcp.PromptArgument{
			{Name: "stack", Description: "e.g. Next.js, Python FastAPI, Go", Required: true},
			{Name: "goal", Description: "What the app should do", Required: true},
		},
	}, buildWeatherApp)
}

func interpretAccuracy(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	station := arg(req, "station_id")
	useCase := arg(req, "use_case")
	if useCase == "" {
		useCase = "general analysis"
	}

	text := `You are analyzing NWS forecast accuracy from the DeltaDaemon API.

## Metrics
- **mean_error (bias):** average forecast − observed. Positive = NWS runs warm.
- **MAE:** typical absolute miss in °F — primary "how wrong" number.
- **RMSE:** penalizes large misses; use when tail risk matters.
- **count:** sample size; prefer ≥30 days before trusting rankings.

## Workflow
1. Call ` + "`list_stations`" + ` to resolve ICAO ids.
2. Call ` + "`get_accuracy_summary`" + ` or ` + "`get_station_accuracy`" + ` for the station and window.
3. For risk: ` + "`get_exceedance`" + ` (miss beyond N°F) and ` + "`get_error_distribution`" + `.
4. For live forecasts: ` + "`get_bias_correction`" + ` on today's NWS number.

## Context
- Use case: ` + useCase + `
`
	if station != "" {
		text += `- Focus station: ` + station + ` — fetch ` + "`get_station_accuracy`" + ` with station_id=` + station + `.\n`
	}
	text += `
Explain results in plain language. Flag low sample counts and whether bias is stable enough to trade on.`

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: text}},
		},
	}, nil
}

func buildWeatherApp(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	stack := arg(req, "stack")
	goal := arg(req, "goal")

	text := `Build a ` + stack + ` application with this goal: ` + goal + `

Use the DeltaDaemon MCP tools (or REST API at https://api.deltadaemon.com/api/v1):
- Auth: run ` + "`deltadaemon-mcp login`" + ` once (credentials in ~/.config/deltadaemon/)
- Read resource ` + "`deltadaemon://docs/overview`" + ` and ` + "`deltadaemon://docs/openapi`" + ` for conventions.
- Prefer typed tools (` + "`get_accuracy_summary`" + `, ` + "`get_exceedance`" + `) over raw ` + "`query_deltadaemon_api`" + `.

Deliver:
1. Project structure; document one-time ` + "`deltadaemon-mcp login`" + ` in README
2. API client module with error handling for 401/402/429
3. Core UI or CLI for the stated goal
4. Example queries for one station (e.g. KLAX) over 90 days
5. Brief README with setup steps`

	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: text}},
		},
	}, nil
}

func arg(req *mcp.GetPromptRequest, name string) string {
	if req.Params == nil || req.Params.Arguments == nil {
		return ""
	}
	return req.Params.Arguments[name]
}
