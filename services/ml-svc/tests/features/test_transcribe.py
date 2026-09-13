"""services/ml-svc/tests/features/test_transcribe.py

Uses hand-written fakes for both components -- this feature only wires the
two together.
"""

from app.features.transcribe import Transcriber


class FakeTranscriber:
    async def transcribe(self, audio, object_key, language, interim):
        yield {"text": f"{object_key}:{audio.decode()}", "is_final": True}


class FakeStorage:
    def get_bytes(self, object_key: str) -> bytes:
        return b"bytes-for-" + object_key.encode()


async def test_transcribe_fetches_then_delegates():
    transcriber = Transcriber(FakeTranscriber(), FakeStorage())
    chunks = [c async for c in transcriber.transcribe("note.m4a", "HINDI", False)]
    assert chunks == [{"text": "note.m4a:bytes-for-note.m4a", "is_final": True}]
