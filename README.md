# LocalID Agent

LocalID Agent is a localhost identity bridge for web applications.

It lets a browser frontend request a cryptographic proof from a local smartcard, eID card, or PKCS#11 device without giving the web app direct access to private keys.

The agent does not authenticate users by itself and does not issue sessions or tokens. Your backend remains the source of truth.

## Prerequisites

- Go 1.25 or newer

## Quick start

```bash
go run ./cmd/localid-agent --config config.example.json
```

The agent binds to `127.0.0.1:17443` by default.

## Smoke tests

```bash
curl http://127.0.0.1:17443/health
curl http://127.0.0.1:17443/status
curl -X POST http://127.0.0.1:17443/sign-challenge \
  -H "Origin: http://localhost:5173" \
  -H "Content-Type: application/json" \
  -d '{"challenge":"YWJj","backend":"http://localhost:8000","purpose":"login","origin":"http://localhost:5173"}'
```

## Configuration

Copy `config.example.json` and adjust:

- `server.host` — defaults to `127.0.0.1`; binding to `0.0.0.0` requires `server.allow_remote_bind: true`
- `security.allowed_origins` — exact browser origins permitted to call the agent
- `security.allowed_backends` — exact backend URLs whose challenges may be signed
- `providers.default` — `mock` for development (PKCS#11 and Belgian eID are stubs in M1–3)

## React example (M4)

End-to-end browser demo: backend challenge → agent signature → backend verification.

Start three terminals:

**Terminal 1 — agent**

```bash
go run ./cmd/localid-agent --config config.example.json
```

**Terminal 2 — mock backend**

```bash
go run ./examples/mock-backend
```

**Terminal 3 — React app**

```bash
cd examples/react
cp .env.example .env
pnpm install
pnpm run dev
```

Open `http://localhost:5173`, confirm the agent status is ready, then click **Authenticate with LocalID**. On success you should see **Mock Dev User**.

The mock backend on `:8000` mirrors the future Laravel M5 `/localid/*` contract. Point `VITE_BACKEND_URL` at Laravel when M5 lands; the React client should need no changes.

See [`examples/react/README.md`](examples/react/README.md) for frontend-only details.

## Development

```bash
go test ./...
go build -o localid-agent ./cmd/localid-agent
```

## Security notes

- Origin and backend values are validated with exact string matching (no wildcards).
- The agent signs a canonical JSON payload (not the raw challenge alone).
- PINs, private keys, and full signatures are never logged.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Agent liveness |
| `/status` | GET | Active provider status |
| `/sign-challenge` | POST | Sign a backend challenge with the configured provider |
