"""services/ml-svc/tests/features/test_verify_technique.py"""

from app.features.verify_technique import TechniqueVerifier


class FakeStorage:
    def get_bytes(self, object_key: str) -> bytes:
        return b""


class FakeVLM:
    def __init__(self, raw: dict) -> None:
        self._raw = raw

    def observe_technique(self, media, object_keys, claimed, craft_id, video_frames):
        return self._raw


async def test_matches_requires_both_the_models_verdict_and_word_overlap():
    # Model says "matches", but observed shares no words with claimed --
    # the cross-check should override it to False.
    raw = {"observed": "wheel throwing", "matches": True, "confidence": 0.9, "explanation": "x"}
    verifier = TechniqueVerifier(FakeVLM(raw), FakeStorage())

    result = await verifier.verify(["clip.mp4"], "resist-dyeing", "craft")
    assert result["matches"] is False


async def test_matches_when_both_agree():
    raw = {"observed": "hand block printing", "matches": True, "confidence": 0.9, "explanation": "x"}
    verifier = TechniqueVerifier(FakeVLM(raw), FakeStorage())

    result = await verifier.verify(["a.jpg"], "hand-block-printing", "craft")
    assert result["matches"] is True
    assert result["claimed"] == "hand-block-printing"
