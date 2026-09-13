"""services/ml-svc/app/models/voice_transcription/

The speech-to-text component. Real mode is an API call (Bhashini), not a
local checkpoint -- proof the "a model can be a call to an external API"
pattern this whole modularization is meant to support already exists in this
codebase, unchanged from before this refactor.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import AsyncIterator, Protocol

from app.config import Config


class VoiceTranscriber(Protocol):
    """Async because real mode streams partial hypotheses over HTTP; mock
    mode is an async generator too, for the same interface."""

    def transcribe(
        self, audio: bytes, object_key: str, language: str, interim: bool
    ) -> AsyncIterator[dict]: ...


@dataclass(frozen=True)
class VoiceTranscriptionConfig:
    url: str = ""
    api_key: str = ""

    @classmethod
    def from_env(cls) -> "VoiceTranscriptionConfig":
        return cls(
            url=os.getenv("ML_SVC_VOICE_TRANSCRIPTION_URL", ""),
            api_key=os.getenv("ML_SVC_VOICE_TRANSCRIPTION_API_KEY", ""),
        )


def build(cfg: Config) -> VoiceTranscriber:
    if cfg.mock_mode:
        from app.models.voice_transcription.mock import MockVoiceTranscriber

        return MockVoiceTranscriber()

    from app.models.voice_transcription.real import RealVoiceTranscriber

    return RealVoiceTranscriber(VoiceTranscriptionConfig.from_env())
