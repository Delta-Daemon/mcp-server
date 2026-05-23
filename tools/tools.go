package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Delta-Daemon/mcp-server/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Registry struct {
	api *client.Client
}

func Register(server *mcp.Server, api *client.Client) {
	r := &Registry{api: api}
	registerAll(server, r)
}

func registerAll(server *mcp.Server, r *Registry) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_stations",
		Description: "List all tracked US weather stations with ICAO id, city, coordinates, timezone, and climate zone. No API key required.",
	}, r.listStations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_accuracy_summary",
		Description: "Aggregate forecast accuracy (MAE, bias, RMSE, sample count) for a station or city over a date window.",
	}, r.getAccuracySummary)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_accuracy_by_city",
		Description: "Rank cities by forecast accuracy. Useful for comparing markets or locations.",
	}, r.getAccuracyByCity)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_station_accuracy",
		Description: "Detailed accuracy stats for one ICAO station (e.g. KLAX), optionally with raw forecast-actual samples.",
	}, r.getStationAccuracy)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_accuracy_by_lead_time",
		Description: "Accuracy broken down by forecast lead-time buckets (0-6h, 6-12h, etc.).",
	}, r.getAccuracyByLeadTime)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_accuracy_by_weather_regime",
		Description: "Accuracy by weather regime (clear, cloudy, precipitation, etc.) for a station.",
	}, r.getAccuracyByWeatherRegime)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_exceedance",
		Description: "Fraction of forecasts exceeding temperature error thresholds (°F). Useful for risk sizing.",
	}, r.getExceedance)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_hit_rate",
		Description: "Forecast hit rate: exact match and within ±1°F / ±2°F for a station or city.",
	}, r.getHitRate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_bias_correction",
		Description: "Apply historical bias correction to a forecast temperature for a station and target date.",
	}, r.getBiasCorrection)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_error_distribution",
		Description: "Histogram of absolute forecast error in 1°F buckets for a station.",
	}, r.getErrorDistribution)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_forecast_actual_pairs",
		Description: "Raw forecast vs observed daily high (or low) pairs for a station or city.",
	}, r.getForecastActual)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_hourly_snapshot",
		Description: "Single NWS model run: hourly predictions and observations around a reference time.",
	}, r.getHourlySnapshot)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_deltadaemon_api",
		Description: "Call any DeltaDaemon GET endpoint under /api/v1. Pass path (e.g. /accuracy/freshness) and optional query params as JSON object. Requires API key for authenticated routes.",
	}, r.queryAPI)
}

type DateWindow struct {
	Days     *int   `json:"days,omitempty" jsonschema:"rolling window in days (default 90, max 365)"`
	DateFrom string `json:"date_from,omitempty" jsonschema:"window start YYYY-MM-DD (overrides days)"`
	DateTo   string `json:"date_to,omitempty" jsonschema:"window end YYYY-MM-DD"`
}

type StationCity struct {
	StationID string `json:"station_id,omitempty" jsonschema:"ICAO station id e.g. KLAX"`
	City      string `json:"city,omitempty" jsonschema:"city name (resolves to station)"`
}

type MetricParam struct {
	Metric string `json:"metric,omitempty" jsonschema:"high (daily max, default) or low (daily min)"`
}

func (d DateWindow) apply(q url.Values) {
	if d.Days != nil {
		q.Set("days", strconv.Itoa(*d.Days))
	}
	if d.DateFrom != "" {
		q.Set("date_from", d.DateFrom)
	}
	if d.DateTo != "" {
		q.Set("date_to", d.DateTo)
	}
}

func (s StationCity) apply(q url.Values) {
	if s.StationID != "" {
		q.Set("station_id", s.StationID)
	}
	if s.City != "" {
		q.Set("city", s.City)
	}
}

func (m MetricParam) apply(q url.Values) {
	if m.Metric != "" {
		q.Set("metric", m.Metric)
	}
}

func jsonResult(raw json.RawMessage) (*mcp.CallToolResult, any, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		buf.Write(raw)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: buf.String()}},
	}, nil, nil
}

func (r *Registry) fetch(ctx context.Context, path string, q url.Values) (*mcp.CallToolResult, any, error) {
	raw, err := r.api.Get(ctx, path, q)
	if err != nil {
		return toolError(err)
	}
	return jsonResult(raw)
}

func toolError(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}, nil, nil
}

func requireAuth(api *client.Client) error {
	if !api.HasAuth() {
		return fmt.Errorf("not logged in — run once in a terminal: deltadaemon-mcp login")
	}
	return nil
}

type listStationsInput struct{}

func (r *Registry) listStations(ctx context.Context, _ *mcp.CallToolRequest, _ listStationsInput) (*mcp.CallToolResult, any, error) {
	return r.fetch(ctx, "/stations/metadata", nil)
}

type accuracySummaryInput struct {
	StationCity
	DateWindow
	MetricParam
}

func (r *Registry) getAccuracySummary(ctx context.Context, _ *mcp.CallToolRequest, in accuracySummaryInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	in.StationCity.apply(q)
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/summary", q)
}

type accuracyByCityInput struct {
	StationCity
	DateWindow
	MetricParam
	MinSamples *int   `json:"min_samples,omitempty" jsonschema:"exclude cities with fewer samples"`
	SortBy     string `json:"sort_by,omitempty" jsonschema:"mae, count, mean_error, city, or rmse"`
}

func (r *Registry) getAccuracyByCity(ctx context.Context, _ *mcp.CallToolRequest, in accuracyByCityInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	in.StationCity.apply(q)
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	if in.MinSamples != nil {
		q.Set("min_samples", strconv.Itoa(*in.MinSamples))
	}
	if in.SortBy != "" {
		q.Set("sort_by", in.SortBy)
	}
	return r.fetch(ctx, "/accuracy/by-city", q)
}

type stationAccuracyInput struct {
	StationID  string `json:"station_id" jsonschema:"required ICAO station id e.g. KLAX"`
	DateWindow
	MetricParam
	IncludeRaw bool `json:"include_raw,omitempty" jsonschema:"include raw forecast-actual samples"`
}

func (r *Registry) getStationAccuracy(ctx context.Context, _ *mcp.CallToolRequest, in stationAccuracyInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	if in.StationID == "" {
		return toolError(fmt.Errorf("station_id is required"))
	}
	q := url.Values{}
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	if in.IncludeRaw {
		q.Set("include_raw", "true")
	}
	path := "/accuracy/by-station/" + url.PathEscape(strings.ToUpper(in.StationID))
	return r.fetch(ctx, path, q)
}

type accuracyByLeadTimeInput struct {
	DateWindow
	MetricParam
}

func (r *Registry) getAccuracyByLeadTime(ctx context.Context, _ *mcp.CallToolRequest, in accuracyByLeadTimeInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/by-lead-time", q)
}

type accuracyByWeatherRegimeInput struct {
	StationID string `json:"station_id,omitempty" jsonschema:"ICAO station id"`
	DateWindow
	MetricParam
}

func (r *Registry) getAccuracyByWeatherRegime(ctx context.Context, _ *mcp.CallToolRequest, in accuracyByWeatherRegimeInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	if in.StationID != "" {
		q.Set("station_id", in.StationID)
	}
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/by-weather-regime", q)
}

type exceedanceInput struct {
	StationID  string `json:"station_id,omitempty" jsonschema:"ICAO station id"`
	Thresholds string `json:"thresholds,omitempty" jsonschema:"comma-separated °F thresholds e.g. 1,2,3,5"`
	DateWindow
	MetricParam
}

func (r *Registry) getExceedance(ctx context.Context, _ *mcp.CallToolRequest, in exceedanceInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	if in.StationID != "" {
		q.Set("station_id", in.StationID)
	}
	if in.Thresholds != "" {
		q.Set("thresholds", in.Thresholds)
	}
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/exceedance", q)
}

type hitRateInput struct {
	StationCity
	DateWindow
	MetricParam
}

func (r *Registry) getHitRate(ctx context.Context, _ *mcp.CallToolRequest, in hitRateInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	in.StationCity.apply(q)
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/hit-rate", q)
}

type biasCorrectionInput struct {
	StationID       string  `json:"station_id" jsonschema:"required ICAO station id"`
	ForecastTemp    float64 `json:"forecast_temp,omitempty" jsonschema:"raw forecast temperature °F (default 75)"`
	ForecastForDate string  `json:"forecast_for_date,omitempty" jsonschema:"target date YYYY-MM-DD"`
	DateWindow
	MetricParam
}

func (r *Registry) getBiasCorrection(ctx context.Context, _ *mcp.CallToolRequest, in biasCorrectionInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	if in.StationID == "" {
		return toolError(fmt.Errorf("station_id is required"))
	}
	q := url.Values{}
	q.Set("station_id", in.StationID)
	if in.ForecastTemp != 0 {
		q.Set("forecast_temp", strconv.FormatFloat(in.ForecastTemp, 'f', -1, 64))
	}
	if in.ForecastForDate != "" {
		q.Set("forecast_for_date", in.ForecastForDate)
	}
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/bias-correction", q)
}

type errorDistributionInput struct {
	StationID string `json:"station_id,omitempty" jsonschema:"ICAO station id"`
	DateWindow
	MetricParam
}

func (r *Registry) getErrorDistribution(ctx context.Context, _ *mcp.CallToolRequest, in errorDistributionInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	if in.StationID != "" {
		q.Set("station_id", in.StationID)
	}
	in.DateWindow.apply(q)
	in.MetricParam.apply(q)
	return r.fetch(ctx, "/accuracy/error-distribution", q)
}

type forecastActualInput struct {
	StationCity
	DateWindow
	Limit *int `json:"limit,omitempty" jsonschema:"max rows (default 1000)"`
}

func (r *Registry) getForecastActual(ctx context.Context, _ *mcp.CallToolRequest, in forecastActualInput) (*mcp.CallToolResult, any, error) {
	if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	in.StationCity.apply(q)
	in.DateWindow.apply(q)
	if in.Limit != nil {
		q.Set("limit", strconv.Itoa(*in.Limit))
	}
	return r.fetch(ctx, "/data/forecast-actual", q)
}

type hourlySnapshotInput struct {
	StationCity
	ReferenceTime string `json:"reference_time,omitempty" jsonschema:"anchor ISO8601 datetime (defaults to now)"`
	LookbackHours *int   `json:"lookback_hours,omitempty" jsonschema:"hours before reference to find a model run (default 6)"`
	UsePublic     bool   `json:"use_public,omitempty" jsonschema:"use rate-limited public endpoint (no API key)"`
}

func (r *Registry) getHourlySnapshot(ctx context.Context, _ *mcp.CallToolRequest, in hourlySnapshotInput) (*mcp.CallToolResult, any, error) {
	path := "/data/hourly-snapshot"
	if in.UsePublic {
		path = "/public/data/hourly-snapshot"
	} else if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	in.StationCity.apply(q)
	if in.ReferenceTime != "" {
		q.Set("reference_time", in.ReferenceTime)
	}
	if in.LookbackHours != nil {
		q.Set("lookback_hours", strconv.Itoa(*in.LookbackHours))
	}
	return r.fetch(ctx, path, q)
}

type queryAPIInput struct {
	Path   string         `json:"path" jsonschema:"API path under /api/v1 e.g. /accuracy/freshness or accuracy/summary"`
	Params map[string]any `json:"params,omitempty" jsonschema:"query parameters as key-value pairs"`
}

func (r *Registry) queryAPI(ctx context.Context, _ *mcp.CallToolRequest, in queryAPIInput) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return toolError(fmt.Errorf("path is required"))
	}
	path := in.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.HasPrefix(path, "/public/") {
		// public routes need no key
	} else if err := requireAuth(r.api); err != nil {
		return toolError(err)
	}
	q := url.Values{}
	for k, v := range in.Params {
		q.Set(k, fmt.Sprint(v))
	}
	return r.fetch(ctx, path, q)
}
