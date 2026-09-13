"""services/ml-svc/tests/features/test_detect_handloom.py"""

from app.features.detect_handloom import HandloomDetector


class FakeHandloom:
    def score(self, image: bytes, object_key: str, declared_thread_count: int) -> dict:
        return {"image": image, "object_key": object_key, "declared_thread_count": declared_thread_count}


class FakeStorage:
    def get_bytes(self, object_key: str) -> bytes:
        return b"bytes-for-" + object_key.encode()


async def test_detect_fetches_then_delegates():
    detector = HandloomDetector(FakeHandloom(), FakeStorage())
    result = await detector.detect("a.jpg", 120)
    assert result == {"image": b"bytes-for-a.jpg", "object_key": "a.jpg", "declared_thread_count": 120}
