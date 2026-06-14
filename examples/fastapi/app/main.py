from __future__ import annotations

import base64
import os
import secrets
import time
from typing import Any

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from app.verifier import VerifyError, verify_proof

CHALLENGE_TTL_SECONDS = 60
ALLOWED_ORIGINS = [
    "http://localhost:5173",
    "http://localhost:5174",
]

app = FastAPI(title="LocalID FastAPI Example", version="0.1.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=False,
    allow_methods=["POST", "OPTIONS"],
    allow_headers=["Content-Type"],
)

_challenges: dict[str, float] = {}


class VerifyBody(BaseModel):
    challenge: str
    backend: str
    origin: str
    purpose: str
    provider: str
    algorithm: str
    signature: str
    certificate: str
    signedAt: str


def _prune_challenges() -> None:
    now = time.time()
    expired = [key for key, expires in _challenges.items() if expires <= now]
    for key in expired:
        _challenges.pop(key, None)


@app.post("/localid/challenge")
async def create_challenge() -> dict[str, str]:
    _prune_challenges()
    challenge = base64.urlsafe_b64encode(secrets.token_bytes(32)).decode("ascii").rstrip("=")
    _challenges[challenge] = time.time() + CHALLENGE_TTL_SECONDS
    return {"challenge": challenge}


@app.post("/localid/verify")
async def verify(request: Request) -> JSONResponse:
    body = await request.json()
    challenge = body.get("challenge", "")

    _prune_challenges()
    expires = _challenges.pop(challenge, None)
    if expires is None or expires <= time.time():
        return JSONResponse({"error": "challenge not found or already used"}, status_code=403)

    try:
        result = verify_proof(body)
        return JSONResponse(result)
    except VerifyError as exc:
        return JSONResponse({"error": str(exc)}, status_code=exc.status_code)


@app.get("/health")
async def health() -> dict[str, Any]:
    return {"ok": True, "name": "LocalID FastAPI Example", "port": int(os.getenv("PORT", "8000"))}
