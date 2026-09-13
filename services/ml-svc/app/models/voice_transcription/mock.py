"""services/ml-svc/app/models/voice_transcription/mock.py

Streams the same chunk shapes Bhashini does, in the same order, from a small
set of canned Hindi sentences -- no network, no weights.
"""

from __future__ import annotations

import asyncio
import hashlib
import random
from typing import AsyncIterator


def _rng(*parts: object) -> random.Random:
    digest = hashlib.sha256("|".join(str(p) for p in parts).encode()).digest()
    return random.Random(int.from_bytes(digest[:8], "big"))


_SENTENCES = [
    "मेरा नाम है और मैं यह काम बीस साल से कर रहा हूँ।",
    "यह कपड़ा हाथ से रंगा गया है।",
    "इसमें प्राकृतिक रंग इस्तेमाल हुए हैं।",
]


class MockVoiceTranscriber:
    async def transcribe(
        self, audio: bytes, object_key: str, language: str, interim: bool
    ) -> AsyncIterator[dict]:
        rng = _rng("asr", object_key, language)
        offset = 0
        for sentence in _SENTENCES[: 1 + rng.randrange(len(_SENTENCES))]:
            duration = 900 + rng.randrange(1200)
            if interim:
                half = len(sentence) // 2
                yield {
                    "text": sentence[:half],
                    "is_final": False,
                    "start_ms": offset,
                    "end_ms": offset + duration // 2,
                    "confidence": 0.0,
                    "language": language,
                }
                await asyncio.sleep(0)
            yield {
                "text": sentence,
                "is_final": True,
                "start_ms": offset,
                "end_ms": offset + duration,
                "confidence": 0.82 + rng.random() * 0.15,
                "language": language,
            }
            offset += duration
