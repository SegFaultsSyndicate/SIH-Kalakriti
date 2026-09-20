"""services/ml-svc/tests/features/test_assess_image_quality.py"""

from app.features.assess_image_quality import ImageQualityAssessor


class FakeImageQuality:
    def assess(self, image: bytes) -> dict:
        return {"passed": False, "issues": [{"code": "BLURRY", "message": "too blurred"}], "blur_score": 12.0}


class FakeStorage:
    def get_bytes(self, object_key: str) -> bytes:
        return b"bytes-for-" + object_key.encode()


async def test_assess_fetches_then_delegates():
    assessor = ImageQualityAssessor(FakeImageQuality(), FakeStorage())
    result = await assessor.assess("a.jpg")
    assert result == {
        "passed": False,
        "issues": [{"code": "BLURRY", "message": "too blurred"}],
        "blur_score": 12.0,
    }
