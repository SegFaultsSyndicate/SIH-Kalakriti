"""services/ml-svc/app/models/vlm/llamacpp.py

A `llama-server` (llama.cpp)-backed `VisionLanguageModel`. Talks to a separate,
long-lived `llama-server` process over its OpenAI-compatible HTTP API
(`/v1/chat/completions`) -- the model itself does not run in this process, so
this module needs neither torch, transformers, bitsandbytes nor PIL (the
biggest weight in the `real` extras). `cv2` is still used for `.mp4`/`.mov`/
`.webm` video-frame sampling, same as `real.py` -- that cost is inherent to
handling video input, not to the model backend.

Why a sidecar instead of an in-process engine: `llama-server` owns its own
GPU-resident state and its own process lifecycle, independent of this
service's `grpc.aio` event loop; a synchronous HTTP call from a
`asyncio.to_thread`-wrapped feature method (see `features/extract_attributes.py`
etc.) is exactly the same shape every other blocking real component already
uses, just over a socket instead of in-process.

Structured output (`response_format: {"type": "json_schema", ...}`) replaces
`real.py`'s `_parse_json` regex extraction for the two calls that expect JSON
back -- a schema-valid reply is now a guarantee, not a hope. This does NOT
replace `features/templates.py`'s allowlist/vocabulary/grounding checks: a
schema-valid answer can still be untrue, and those checks catch that
regardless of which backend produced the answer.
"""

from __future__ import annotations

import base64
import json
import logging

from app.models.vlm import VLMConfig
from app.models.vlm.prompts import EXTRACT_PROMPT, POLISH_PROMPT, technique_prompt

log = logging.getLogger(__name__)

# Generation can genuinely take tens of seconds on a small GPU; matches the
# timeout voice_transcription/real.py already uses for its own model call.
_REQUEST_TIMEOUT_S = 120.0

_ATTRIBUTES_SCHEMA = {
    "type": "object",
    "properties": {
        "craft_id": {"type": "string"},
        "material": {"type": "string"},
        "technique": {"type": "string"},
        "colours": {"type": "array", "items": {"type": "string"}},
        "motifs": {"type": "array", "items": {"type": "string"}},
        "confidence": {
            "type": "object",
            "properties": {
                "craft_id": {"type": "number"},
                "material": {"type": "number"},
                "technique": {"type": "number"},
                "colours": {"type": "number"},
                "motifs": {"type": "number"},
            },
            "required": ["craft_id", "material", "technique", "colours", "motifs"],
        },
    },
    "required": ["craft_id", "material", "technique", "colours", "motifs", "confidence"],
}

_TECHNIQUE_SCHEMA = {
    "type": "object",
    "properties": {
        "observed": {"type": "string"},
        "matches": {"type": "boolean"},
        "confidence": {"type": "number"},
        "explanation": {"type": "string"},
    },
    "required": ["observed", "matches", "confidence", "explanation"],
}


class LlamaCppVisionLanguageModel:
    def __init__(self, cfg: VLMConfig) -> None:
        self._base_url = cfg.llamacpp_base_url.rstrip("/")

    def _chat(
        self,
        images: list[bytes],
        prompt: str,
        json_schema: dict | None = None,
        max_tokens: int = 512,
    ) -> str:
        import httpx

        content: list[dict] = [{"type": "text", "text": prompt}]
        for image in images:
            encoded = base64.b64encode(image).decode("ascii")
            content.append({"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{encoded}"}})

        body = {
            "messages": [{"role": "user", "content": content}],
            # Greedy, to match real.py's do_sample=False -- server defaults may
            # apply the checkpoint's own generation_config otherwise.
            "temperature": 0,
            "max_tokens": max_tokens,
        }
        if json_schema is not None:
            body["response_format"] = {
                "type": "json_schema",
                "json_schema": {"name": "response", "schema": json_schema},
            }

        response = httpx.post(
            f"{self._base_url}/v1/chat/completions", json=body, timeout=_REQUEST_TIMEOUT_S
        )
        response.raise_for_status()
        return response.json()["choices"][0]["message"]["content"]

    def propose_attributes(
        self,
        images: list[bytes],
        object_keys: list[str],
        allowlist: list[str],
        declared_craft_id: str,
        hint: str,
        vocab_hint: str,
        craft_vocab: dict[str, dict[str, list[str]]],
    ) -> dict:
        raw = self._chat(
            images,
            EXTRACT_PROMPT.format(
                allowlist="\n".join(allowlist),
                declared=declared_craft_id or "(not stated)",
                hint=hint or "(nothing)",
                vocab_hint=vocab_hint,
            ),
            json_schema=_ATTRIBUTES_SCHEMA,
        )
        parsed = _parse_json(raw)
        return {
            "craft_id": str(parsed.get("craft_id") or ""),
            "material": str(parsed.get("material") or ""),
            "technique": str(parsed.get("technique") or ""),
            "colours": parsed.get("colours"),
            "motifs": parsed.get("motifs"),
            "confidence": parsed.get("confidence") or {},
            # Same as real.py: not readable from a photograph without a
            # reference object, so absent rather than guessed.
            "dimensions": None,
            "dimensions_confidence": 0.0,
        }

    def observe_technique(
        self,
        media: list[bytes],
        object_keys: list[str],
        claimed: str,
        craft_id: str,
        video_frames: int,
    ) -> dict:
        frames = []
        for data, key in zip(media, object_keys):
            if key.lower().endswith((".mp4", ".mov", ".webm")):
                frames.extend(_video_frames(data, video_frames))
            else:
                frames.append(data)

        raw = self._chat(
            frames[:video_frames],
            technique_prompt(craft_id, claimed),
            json_schema=_TECHNIQUE_SCHEMA,
            max_tokens=256,
        )
        parsed = _parse_json(raw)
        return {
            "observed": str(parsed.get("observed") or ""),
            "matches": bool(parsed.get("matches")),
            "confidence": _confidence(parsed, "confidence"),
            "explanation": str(parsed.get("explanation") or ""),
        }

    def polish(self, template: str, facts: dict, language: str, max_chars: int) -> str:
        raw = self._chat(
            [],
            POLISH_PROMPT.format(template=template, language=language or "ENGLISH", max_chars=max_chars or 900),
            max_tokens=256,
        )
        return raw.strip()


def _parse_json(raw: str) -> dict:
    """Structured output should always hand back a clean JSON object, but a
    server running without `response_format` support (or serving a non-VLM
    fallback) would not -- keep the same defensive parse `real.py` uses
    rather than assuming the guarantee always holds."""
    import re

    match = re.search(r"\{.*\}", raw, re.S)
    if not match:
        log.warning("llama-server reply carried no json", extra={"reply": raw[:200]})
        return {}
    try:
        return json.loads(match.group(0))
    except json.JSONDecodeError:
        log.warning("llama-server reply was not valid json", extra={"reply": raw[:200]})
        return {}


def _confidence(source: dict, key: str) -> float:
    try:
        return max(0.0, min(1.0, float(source.get(key, 0.0))))
    except (TypeError, ValueError):
        return 0.0


def _video_frames(data: bytes, count: int) -> list[bytes]:
    """N frames spread evenly across a clip, as JPEG-encoded bytes (this
    backend sends raw bytes over HTTP, not PIL objects)."""
    import tempfile

    import cv2

    with tempfile.NamedTemporaryFile(suffix=".mp4") as handle:
        handle.write(data)
        handle.flush()
        capture = cv2.VideoCapture(handle.name)
        try:
            total = int(capture.get(cv2.CAP_PROP_FRAME_COUNT)) or 1
            frames = []
            for i in range(count):
                capture.set(cv2.CAP_PROP_POS_FRAMES, int(i * total / count))
                ok, frame = capture.read()
                if ok:
                    ok, encoded = cv2.imencode(".jpg", frame)
                    if ok:
                        frames.append(encoded.tobytes())
            return frames
        finally:
            capture.release()
