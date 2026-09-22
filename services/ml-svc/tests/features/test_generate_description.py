"""services/ml-svc/tests/features/test_generate_description.py"""

from app.features.generate_description import DescriptionGenerator


class FakeVLM:
    def __init__(self, rewrite=None, raises=False) -> None:
        self._rewrite = rewrite
        self._raises = raises

    def polish(self, template, facts, language, max_chars):
        if self._raises:
            raise RuntimeError("boom")
        return self._rewrite if self._rewrite is not None else template


def _attrs(**kwargs):
    return {k: (v, 0.9) for k, v in kwargs.items()}


async def test_uses_the_template_when_polish_hallucinates_an_unrelated_claim():
    generator = DescriptionGenerator(FakeVLM(rewrite="A beautiful silk piece, handcrafted with care."))
    attributes = _attrs(material="cotton")

    result = await generator.generate(attributes, "ajrakh-block-printing", "ENGLISH", "", 0)
    assert "silk" not in result["description"]
    assert "cotton" in result["description"]


async def test_uses_the_template_when_polish_raises():
    generator = DescriptionGenerator(FakeVLM(raises=True))
    result = await generator.generate(_attrs(material="cotton"), "ajrakh-block-printing", "ENGLISH", "", 0)
    assert "cotton" in result["description"]


async def test_only_cites_attributes_it_was_given():
    generator = DescriptionGenerator(FakeVLM())
    attributes = _attrs(material="cotton", technique="hand-block-printing")

    result = await generator.generate(attributes, "ajrakh-block-printing", "ENGLISH", "my story", 200)
    assert result["title"] and result["description"]
    assert len(result["description"]) <= 200
    assert set(result["attribute_keys_used"]) <= set(attributes)
    assert "my story" in result["description"]
