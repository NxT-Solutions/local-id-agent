from __future__ import annotations

import base64
import json
import re
from datetime import datetime, timezone

from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding
from cryptography.hazmat.primitives.asymmetric.rsa import RSAPublicKey
from cryptography.x509 import load_der_x509_certificate

BASE64URL_RE = re.compile(r"^[A-Za-z0-9_-]+$")
CHALLENGE_MAX_AGE_SECONDS = 60
EXPECTED_BACKEND = "http://localhost:8000"
ALLOWED_ORIGINS = frozenset({"http://localhost:5173", "http://localhost:5174"})


class VerifyError(Exception):
    def __init__(self, message: str, status_code: int = 403) -> None:
        super().__init__(message)
        self.status_code = status_code


def build_canonical_payload(
    *,
    backend: str,
    challenge: str,
    origin: str,
    purpose: str,
    timestamp: str,
) -> bytes:
    if not challenge or not BASE64URL_RE.fullmatch(challenge):
        raise VerifyError("challenge must be valid base64url", 400)

    payload = {
        "backend": backend,
        "challenge": challenge,
        "origin": origin,
        "purpose": purpose,
        "timestamp": timestamp,
    }
    return json.dumps(payload, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def verify_proof(body: dict[str, str], *, now: datetime | None = None) -> dict[str, object]:
    now = now or datetime.now(timezone.utc)

    challenge = body.get("challenge", "")
    backend = body.get("backend", "")
    origin = body.get("origin", "")
    purpose = body.get("purpose", "")
    algorithm = body.get("algorithm", "")
    signature_b64 = body.get("signature", "")
    certificate_b64 = body.get("certificate", "")
    signed_at = body.get("signedAt", "")

    if algorithm != "RS256":
        raise VerifyError("unsupported algorithm")
    if purpose != "login":
        raise VerifyError("purpose is not allowed")
    if backend != EXPECTED_BACKEND:
        raise VerifyError("backend is not allowed")
    if origin not in ALLOWED_ORIGINS:
        raise VerifyError("origin is not allowed")
    if not certificate_b64:
        raise VerifyError("certificate is required", 400)

    try:
        signed_time = datetime.fromisoformat(signed_at.replace("Z", "+00:00"))
    except ValueError as exc:
        raise VerifyError("signedAt must be RFC3339", 400) from exc

    if signed_time.tzinfo is None:
        signed_time = signed_time.replace(tzinfo=timezone.utc)

    age = (now - signed_time.astimezone(timezone.utc)).total_seconds()
    if age < 0 or age > CHALLENGE_MAX_AGE_SECONDS:
        raise VerifyError("challenge timestamp is stale or invalid")

    payload = build_canonical_payload(
        backend=backend,
        challenge=challenge,
        origin=origin,
        purpose=purpose,
        timestamp=signed_at,
    )

    try:
        padding = "=" * (-len(signature_b64) % 4)
        signature = base64.urlsafe_b64decode(signature_b64 + padding)
    except Exception as exc:
        raise VerifyError("signature must be valid base64url", 400) from exc

    try:
        cert_der = base64.b64decode(certificate_b64)
        cert = load_der_x509_certificate(cert_der)
    except Exception as exc:
        raise VerifyError("invalid certificate", 403) from exc

    public_key = cert.public_key()
    if not isinstance(public_key, RSAPublicKey):
        raise VerifyError("certificate public key is not RSA")

    try:
        public_key.verify(
            signature,
            payload,
            padding.PKCS1v15(),
            hashes.SHA256(),
        )
    except Exception as exc:
        raise VerifyError("signature verification failed") from exc

    return {
        "success": True,
        "user": {"id": "mock-user", "name": "Mock Dev User"},
    }
