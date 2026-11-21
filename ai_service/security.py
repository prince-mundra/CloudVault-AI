from __future__ import annotations

import hashlib
import hmac
import time

from fastapi import HTTPException, Request

TIMESTAMP_HEADER = "X-CloudVault-Timestamp"
SIGNATURE_HEADER = "X-CloudVault-Signature"
MAX_CLOCK_SKEW_SECONDS = 60


def request_signature(secret: str, timestamp: str, method: str, path: str, body: bytes) -> str:
    body_hash = hashlib.sha256(body).hexdigest()
    message = f"{timestamp}\n{method.upper()}\n{path}\n{body_hash}".encode()
    return hmac.new(secret.encode("utf-8"), message, hashlib.sha256).hexdigest()


async def require_service_signature(request: Request) -> None:
    secret = request.app.state.settings.internal_secret
    if len(secret.encode("utf-8")) < 32:
        raise HTTPException(status_code=503, detail="Internal AI service authentication is not configured")

    timestamp = request.headers.get(TIMESTAMP_HEADER, "")
    supplied = request.headers.get(SIGNATURE_HEADER, "")
    try:
        timestamp_value = int(timestamp)
    except (TypeError, ValueError):
        raise HTTPException(status_code=401, detail="Invalid service authentication") from None
    if abs(int(time.time()) - timestamp_value) > MAX_CLOCK_SKEW_SECONDS:
        raise HTTPException(status_code=401, detail="Invalid service authentication")

    body = await request.body()
    expected = request_signature(secret, timestamp, request.method, request.url.path, body)
    if not hmac.compare_digest(expected, supplied):
        raise HTTPException(status_code=401, detail="Invalid service authentication")
