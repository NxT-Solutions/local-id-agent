# Agent configuration

## Quick start

```bash
cd services/agent
go run ./cmd/localid-agent --config config.example.json
```

The agent binds to `127.0.0.1:17443` by default.

## Smoke tests

```bash
curl http://127.0.0.1:17443/health
curl http://127.0.0.1:17443/status
curl http://127.0.0.1:17443/openapi.json   # requires server.dev_mode: true
curl -X POST http://127.0.0.1:17443/sign-challenge \
  -H "Origin: http://localhost:5173" \
  -H "Content-Type: application/json" \
  -d '{"challenge":"YWJj","backend":"http://localhost:8000","purpose":"login","origin":"http://localhost:5173"}'
```

## Configuration file

Copy `services/agent/config.example.json` and adjust:

- `server.host` — defaults to `127.0.0.1`; binding to `0.0.0.0` requires `server.allow_remote_bind: true`
- `server.dev_mode` — when `true`, serves OpenAPI at `GET /openapi.json` and `GET /openapi.yaml` (disable in production)
- `security.allowed_origins` — exact browser origins permitted to call the agent
- `security.allowed_backends` — exact backend URLs whose challenges may be signed
- `providers.default` — choose `mock`, `pkcs11`, or `belgian_eid`
- `providers.pkcs11.pin_prompt = "terminal"` reads PIN from `LOCALID_PKCS11_PIN` when token login is required
- `LOCALID_BEID_PKCS11_MODULE` can override Belgian eID auto module detection

## Mock backend

For local development and demos, run the mock backend alongside the agent:

```bash
cd services/agent
go run ./cmd/mock-backend
```

The mock backend listens on `:8000` and mirrors the future Laravel M5 `/localid/*` contract.

## Development

```bash
cd services/agent
go test ./...
go build -o localid-agent ./cmd/localid-agent
```

From the repository root:

```bash
pnpm run test:go
pnpm run build:sidecar
```
