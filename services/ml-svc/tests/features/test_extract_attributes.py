"""services/ml-svc/tests/features/test_extract_attributes.py

A hand-written fake vlm -- exercising this feature's closed-vocabulary
enforcement does not need app.models.vlm at all.
"""

from app.features.extract_attributes import AttributeExtractor


class FakeStorage:
    def get_bytes(self, object_key: str) -> bytes:
        return b""


class FakeVLM:
    def __init__(self, raw: dict) -> None:
        self._raw = raw

    def propose_attributes(self, images, object_keys, allowlist, declared_craft_id, hint, vocab_hint, craft_vocab):
        return self._raw


def _extractor(raw: dict, craft_allowlist_path) -> AttributeExtractor:
    return AttributeExtractor(FakeVLM(raw), FakeStorage(), craft_allowlist_path)


async def test_rejects_a_craft_outside_the_ontology(tmp_path):
    allowlist = tmp_path / "crafts.csv"
    allowlist.write_text("ajrakh-block-printing\n")
    raw = {"craft_id": "not-a-real-craft", "material": "", "technique": "", "confidence": {}}

    result = await _extractor(raw, allowlist).extract(["a.jpg"], "ajrakh-block-printing", "")
    assert result["craft_id"][0] == "ajrakh-block-printing"  # falls back to declared


async def test_rejects_a_material_outside_the_crafts_vocabulary(tmp_path):
    allowlist = tmp_path / "crafts.csv"
    allowlist.write_text(
        "code,display_name,parent_code,gi_registration_no,techniques,materials\n"
        "ajrakh-block-printing,x,,,hand-block-printing,cotton\n"
    )
    raw = {
        "craft_id": "ajrakh-block-printing", "material": "silk", "technique": "hand-block-printing",
        "confidence": {"craft_id": 0.9, "material": 0.9, "technique": 0.9},
    }

    result = await _extractor(raw, allowlist).extract(["a.jpg"], "", "")
    assert result["material"] == ("", 0.0)  # rejected: not in this craft's vocabulary
    assert result["technique"] == ("hand-block-printing", 0.9)  # accepted: is in vocabulary


async def test_confidence_is_zero_when_the_field_ends_up_empty(tmp_path):
    allowlist = tmp_path / "crafts.csv"
    allowlist.write_text("ajrakh-block-printing\n")
    raw = {"craft_id": "", "material": "", "technique": "", "confidence": {"craft_id": 0.9}}

    result = await _extractor(raw, allowlist).extract(["a.jpg"], "", "")
    assert result["craft_id"] == ("", 0.0)
