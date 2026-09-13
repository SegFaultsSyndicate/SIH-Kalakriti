"""services/ml-svc/app/models/voice_transcription/real.py

Bhashini's streaming ASR endpoint. `httpx` is imported here, not at module
scope in `__init__.py`, so mock mode never needs the package installed.
"""

from __future__ import annotations

import base64
import json
import logging
from typing import AsyncIterator

from app.models.voice_transcription import VoiceTranscriptionConfig

log = logging.getLogger(__name__)


class RealVoiceTranscriber:
    def __init__(self, cfg: VoiceTranscriptionConfig) -> None:
        self._cfg = cfg

    async def transcribe(
        self, audio: bytes, object_key: str, language: str, interim: bool
    ) -> AsyncIterator[dict]:
        import httpx

        headers = {"Authorization": self._cfg.api_key, "Content-Type": "application/json"}
        payload = {
            "config": {"language": {"sourceLanguage": _bhashini_code(language)}, "interimResults": interim},
            "audio": [{"audioContent": base64.b64encode(audio).decode()}],
        }

        async with httpx.AsyncClient(timeout=120) as client:
            async with client.stream("POST", self._cfg.url, json=payload, headers=headers) as response:
                response.raise_for_status()
                offset = 0
                async for line in response.aiter_lines():
                    if not line.strip():
                        continue
                    chunk = _parse_json(line)
                    text = str(chunk.get("text") or "")
                    if not text:
                        continue
                    end = int(chunk.get("end_ms") or offset + 1000)
                    yield {
                        "text": text,
                        "is_final": bool(chunk.get("is_final", True)),
                        "start_ms": int(chunk.get("start_ms") or offset),
                        "end_ms": end,
                        "confidence": _confidence(chunk, "confidence"),
                        "language": language,
                    }
                    offset = end


def _parse_json(raw: str) -> dict:
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        log.warning("bhashini line was not valid json", extra={"reply": raw[:200]})
        return {}


def _confidence(source: dict, key: str) -> float:
    try:
        return max(0.0, min(1.0, float(source.get(key, 0.0))))
    except (TypeError, ValueError):
        return 0.0


def _bhashini_code(language: str) -> str:
    """Eighth Schedule name to the ISO code Bhashini expects."""
    return {
        "HINDI": "hi", "BENGALI": "bn", "ASSAMESE": "as", "GUJARATI": "gu",
        "KANNADA": "kn", "MALAYALAM": "ml", "MARATHI": "mr", "ODIA": "or",
        "PUNJABI": "pa", "TAMIL": "ta", "TELUGU": "te", "URDU": "ur",
        "MAITHILI": "mai", "NEPALI": "ne", "SANSKRIT": "sa", "SINDHI": "sd",
        "KONKANI": "kok", "DOGRI": "doi", "BODO": "brx",
        "KASHMIRI": "ks", "ENGLISH": "en",
    }.get(language.upper(), "hi")
