# FastAPI example

Python [FastAPI](https://fastapi.tiangolo.com/) backend implementing the LocalID `/localid/challenge` and `/localid/verify` contract.

## Prerequisites

- Python 3.12+
- LocalID Agent running (`services/agent`)
- Add `http://localhost:5173` and/or `http://localhost:5174` to agent `allowed_origins`

## Setup

```bash
cd examples/fastapi
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

## Run

```bash
uvicorn app.main:app --reload --port 8000
```

## Test with React or Vue example

```bash
# Terminal 1 — agent
cd services/agent && go run ./cmd/localid-agent --config config.example.json

# Terminal 2 — this API
cd examples/fastapi && source .venv/bin/activate && uvicorn app.main:app --reload --port 8000

# Terminal 3 — frontend (from repo root)
pnpm run dev:react   # or: pnpm run dev:vue
```

Set `VITE_BACKEND_URL=http://localhost:8000` in the frontend `.env`.
