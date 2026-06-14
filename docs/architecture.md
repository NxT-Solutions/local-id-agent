# Architecture

LocalID Agent is a localhost identity bridge for web applications. It lets a browser frontend request a cryptographic proof from a local smartcard, eID card, or PKCS#11 device without giving the web app direct access to private keys.

The agent does not authenticate users by itself and does not issue sessions or tokens. Your backend remains the source of truth.

## Monorepo layout

| Path | Description |
|------|-------------|
| `proto/localid/v1/` | Protobuf API definitions (source of truth) |
| `services/agent/` | Go agent, mock backend, and config |
| `packages/localid-client` | Shared TypeScript client (`@rqc-icu/localid-client`) |
| `apps/desktop` | Tauri 2 desktop app |
| `examples/react` | Browser demo using the shared client |

## Auth flow

1. Frontend calls backend `POST /localid/challenge` to get a one-time challenge.
2. Frontend calls agent `POST /sign-challenge` with challenge, backend URL, purpose, and origin.
3. Agent validates origin/backend, signs a canonical JSON payload with the configured provider.
4. Frontend sends proof to backend `POST /localid/verify`.
5. Backend verifies signature, challenge freshness, and issues a session.

## Code generation (Protobuf + Buf)

API message types are defined once in `proto/localid/v1/protocol.proto` and generated for Go and TypeScript:

| Output | Generator | Path |
|--------|-----------|------|
| Go structs | `protoc-gen-go` | `services/agent/internal/protocol/pb/` |
| TypeScript types | `@bufbuild/protoc-gen-es` | `packages/localid-client/src/generated/` |

```bash
pnpm generate          # or: buf generate
```

The Go agent uses type aliases in `services/agent/internal/protocol/types.go`; the TS client re-exports generated types from `packages/localid-client/src/types.ts`. Client-only types (backend verify flow, UI state) remain hand-written in `types.ts`.

Turborepo runs `generate` before `build` and `test`. After changing `.proto` files, run `pnpm generate` and commit the generated outputs.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Agent liveness |
| `/status` | GET | Active provider status |
| `/sign-challenge` | POST | Sign a backend challenge with the configured provider |

## Security notes

- Origin and backend values are validated with exact string matching (no wildcards).
- The agent signs a canonical JSON payload (not the raw challenge alone).
- PINs, private keys, and full signatures are never logged.
