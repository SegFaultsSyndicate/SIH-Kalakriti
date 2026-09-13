"""services/ml-svc/app/features/transcribe.py

The Transcribe RPC's business logic: fetch the audio bytes, hand them to the
voice-transcription component, stream its chunks back unchanged.
"""

from __future__ import annotations

import asyncio
from typing import AsyncIterator

from app.models.storage import ObjectStorage
from app.models.voice_transcription import VoiceTranscriber


class Transcriber:
    def __init__(self, transcriber: VoiceTranscriber, storage: ObjectStorage) -> None:
        self._transcriber = transcriber
        self._storage = storage

    async def transcribe(self, object_key: str, language: str, interim: bool) -> AsyncIterator[dict]:
        audio = await asyncio.to_thread(self._storage.get_bytes, object_key)
        async for chunk in self._transcriber.transcribe(audio, object_key, language, interim):
            yield chunk
