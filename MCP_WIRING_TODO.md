# MCP ↔ DeltaDaemon wiring TODO

Handoff doc for work across **two repos**:

| Repo | Path (local) | Remote |
|------|----------------|--------|
| MCP server | `~/Desktop/mcp-server` | `github.com/Delta-Daemon/mcp-server` |
| Main app (API + frontend) | `~/Desktop/Deltadaemon` | (your org repo) |

**Goal:** A user can install `deltadaemon-mcp`, run `setup`, sign in via browser OAuth, paste one JSON snippet into Cursor/Claude, and call authenticated accuracy tools.

---

## Already implemented (do not redo)

### mcp-server
- [x] Browser OAuth login (`auth/oauth.go`) — local callback on `127.0.0.1`, hits `GET /auth/mcp/login`
- [x] `deltadaemon-mcp setup` — login + print MCP config (`auth/setup.go`)
- [x] `deltadaemon-mcp login` defaults to OAuth; `--password` / `--api-key` kept as fallbacks
- [x] Install script (`scripts/install.sh`) — release tarball or `go install`
- [x] Makefile release targets (`Makefile`)
- [x] OpenAPI docs include `/auth/mcp/login` (`resources/openapi.yaml`)

### Deltadaemon / delta-daemon-api (local only — **not deployed**)
- [x] `GET /auth/mcp/login` handler (`delta-daemon-api/handlers/auth_mcp.go`)
- [x] OAuth callback redirects to loopback when MCP cookies present (`auth_oauth.go` → `mcpRedirectDestination`)
- [x] Route registered in `delta-daemon-api/main.go`

---

## Blockers (must ship before OAuth works in prod)

Production API currently returns **404** for `/auth/oauth/*` and `/auth/mcp/login`. OAuth env vars must also be set.

- [ ] **Deploy API with MCP auth changes**
  - Files: `handlers/auth_mcp.go`, `handlers/auth_oauth.go` (session ID capture), `main.go` route
  - Verify after deploy:
    ```bash
    curl -sI "https://api.deltadaemon.com/auth/mcp/login?provider=google&redirect_uri=http://127.0.0.1:8765/callback&state=test" | head -5
    # expect 302, not 404
    ```
- [ ] **Confirm OAuth env vars in prod** (see `Deltadaemon/.env.example`)
  - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
  - `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`
  - `API_BASE_URL=https://api.deltadaemon.com`
  - `FRONTEND_URL=https://deltadaemon.com`
  - `COOKIE_SECURE=true`, `COOKIE_DOMAIN` if using shared parent domain

---

## Deltadaemon repo tasks

### API docs & checklist sync
- [ ] Add `/auth/mcp/login` to `delta-daemon-api/openapi.yaml` (copy from `mcp-server/resources/openapi.yaml`)
- [ ] Add `/auth/mcp/login` to `frontend/public/openapi.yaml` (keep in sync)
- [ ] Add row to `delta-daemon-api/API_CHECKLIST.md`:
  ```
  GET | /auth/mcp/login | GetMCPLogin
  ```

### Tests
- [ ] Unit tests for `isLoopbackRedirectURI` and `mcpRedirectDestination` in `handlers/auth_mcp_test.go`
- [ ] Integration test: mock OAuth callback with `dd_mcp_redirect_uri` cookie → asserts redirect URL contains `session=` and `state=`
- [ ] Optional: router test in `main_test.go` that `/auth/mcp/login` returns 400 without params, 302 with valid loopback URI (may need OAuth env mocked)

### Security review (before prod)
- [ ] Confirm loopback-only redirect URIs (`127.0.0.1`, `localhost`, `[::1]`) — already in `isLoopbackRedirectURI`
- [ ] Session token passed in query string over localhost only — acceptable for v1; consider one-time exchange code in v2
- [ ] MCP cookies (`dd_mcp_redirect_uri`, `dd_mcp_client_state`) are HttpOnly, 5-min TTL, cleared after use
- [ ] OAuth `state` validated on callback (existing `dd_oauth_state` check)

### Ops / monitoring
- [ ] Add Datadog synthetic or smoke check for `GET /auth/mcp/login` (expect redirect, not 5xx)
- [ ] Log MCP login attempts at info level (provider + loopback port, not full session)

### Repo hygiene
- [ ] Update `CHANGELOG.md` MCP section — OAuth browser login, `/auth/mcp/login`, standalone mcp-server repo
- [ ] Update `.env.example` MCP section — replace “clone and build” with install script + `deltadaemon-mcp setup`
- [ ] Commit + PR the API changes (currently only local)

---

## mcp-server repo tasks

### Releases & install path
- [ ] Add `.github/workflows/release.yml`:
  - Trigger on tag `v*`
  - Build `deltadaemon-mcp_{darwin,linux}_{amd64,arm64}`
  - Tar.gz each binary (install script expects `{BINARY}_{os}_{arch}.tar.gz`)
  - Upload GitHub release assets
- [ ] Tag first release (`v0.1.0` or similar) so `scripts/install.sh` release download works
- [ ] Verify install script end-to-end:
  ```bash
  curl -fsSL .../scripts/install.sh | sh
  deltadaemon-mcp setup
  ```

### CI
- [ ] Add `.github/workflows/test.yml` — `go test ./...` + `go vet ./...` on PR

### Docs
- [ ] README: note prod OAuth requires deployed API (link to deploy checklist above)
- [ ] Optional: `docs/CURSOR.md` and `docs/CLAUDE.md` with screenshots / restart instructions

### Error UX (after API is live)
- [ ] If `/auth/mcp/login` returns 404, print actionable message: “API too old — MCP OAuth not deployed yet”
- [ ] If OAuth provider not configured (503), suggest `--provider github` or `--api-key`

### Optional improvements
- [ ] `setup --write-config` — merge into `~/.cursor/mcp.json` (ask before overwriting)
- [ ] Provider picker when `--provider` omitted (interactive: Google vs GitHub)
- [ ] Version string in `deltadaemon-mcp status` (`-ldflags` in Makefile already stubbed)

---

## Frontend / site tasks (Deltadaemon)

No frontend code exists for MCP yet.

- [ ] Add `/mcp` or `/developers/mcp` page on deltadaemon.com:
  - One-line install: `curl -fsSL ... | sh`
  - Three steps: install → `deltadaemon-mcp setup` → paste JSON into Cursor/Claude
  - Download links to GitHub releases (darwin arm64/amd64, linux)
  - Link to API key docs as fallback
- [ ] Add nav/footer link (“MCP” or “Cursor integration”)
- [ ] Optional: authenticated dashboard card — “Connect Cursor” with copy-paste config using detected OS

---

## Cross-repo verification checklist

Run after API deploy + mcp-server release.

### Local (API on `:8105`)
```bash
# Terminal 1 — API
cd Deltadaemon/delta-daemon-api
# ensure OAuth env vars set, then run API

# Terminal 2 — MCP
cd mcp-server
go build -o deltadaemon-mcp .
./deltadaemon-mcp login --api-base http://localhost:8105/api/v1
./deltadaemon-mcp status
./deltadaemon-mcp setup
```

- [ ] Browser opens, Google/GitHub sign-in succeeds
- [ ] Localhost callback shows “Signed in” page
- [ ] `~/.config/deltadaemon/credentials.json` created (mode 0600)
- [ ] `deltadaemon-mcp status` shows email + `session (saved)`

### Cursor / Claude
- [ ] Add printed JSON to MCP config
- [ ] Restart editor
- [ ] `list_stations` works without auth
- [ ] `get_accuracy_summary` works with saved session (requires active plan)
- [ ] Unauthenticated error says `run: deltadaemon-mcp setup`

### Production
```bash
deltadaemon-mcp login   # default API base
deltadaemon-mcp status
```
- [ ] Same flow against `https://api.deltadaemon.com`

---

## Contract between repos (keep in sync)

### MCP client → API

**Start OAuth**
```
GET {AuthBase}/auth/mcp/login
  ?provider=google|github
  &redirect_uri=http://127.0.0.1:{port}/callback
  &state={random_hex}
```
→ 302 to Google/GitHub

**After OAuth (API → browser → CLI callback)**
```
GET http://127.0.0.1:{port}/callback?session={dd_session_id}&state={client_state}
```
CLI verifies `state`, calls `GET /auth/me` with session cookie, saves to credentials file.

**Auth base URL**
- MCP derives from `DELTADAEMON_API_BASE` or default `https://api.deltadaemon.com/api/v1`
- Strips `/api/v1` → `https://api.deltadaemon.com` (`auth/store.go` → `AuthBase()`)

### Files that must stay aligned
| Concern | mcp-server | Deltadaemon |
|---------|------------|-------------|
| MCP login endpoint path | `auth/oauth.go` → `/auth/mcp/login` | `handlers/auth_mcp.go`, `main.go` |
| OpenAPI | `resources/openapi.yaml` | `delta-daemon-api/openapi.yaml`, `frontend/public/openapi.yaml` |
| Install docs | `README.md`, `scripts/install.sh` | `.env.example`, site `/mcp` page |
| Session cookie name | `dd_session` in `client/client.go` | `handlers/auth.go` `setSessionCookie` |

---

## Suggested PR order

1. **Deltadaemon PR:** API MCP OAuth endpoint + tests + openapi/checklist + CHANGELOG
2. **Deploy API** to staging/prod
3. **mcp-server PR:** CI + release workflow + any error-message polish
4. **Tag mcp-server release**
5. **Deltadaemon PR:** site `/mcp` page + `.env.example` / docs updates
6. **End-to-end verify** using checklist above

---

## Out of scope (later)

- MCP OAuth 2.1 / remote HTTP transport (Cursor native URL auth)
- One-time code exchange instead of session in query string
- Windows install (.exe / scoop / winget)
- npm wrapper (`npx deltadaemon-mcp`) — not needed if release binaries work
