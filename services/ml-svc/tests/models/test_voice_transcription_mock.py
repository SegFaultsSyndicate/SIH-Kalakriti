"""services/ml-svc/tests/models/test_voice_transcription_mock.py"""

from app.config import Config
from app.models import voice_transcription


async def test_streams_final_chunks_in_order():
    backend = voice_transcription.build(Config(mock_mode=True))
    chunks = [c async for c in backend.transcribe(b"", "note.m4a", "HINDI", True)]

    assert chunks
    finals = [c for c in chunks if c["is_final"]]
    assert finals == sorted(finals, key=lambda c: c["start_ms"])
    for chunk in finals:
        assert chunk["end_ms"] > chunk["start_ms"]
        assert 0.0 <= chunk["confidence"] <= 1.0
