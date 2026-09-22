"""services/ml-svc/tests/features/test_translate.py"""

from app.features.translate import Translator


class FakeTranslationModel:
    def translate(self, title, description, highlights, source_language, target_language):
        return {
            "title": f"{target_language}:{title}",
            "description": f"{target_language}:{description}",
            "highlights": [f"{target_language}:{h}" for h in highlights],
        }


async def test_translate_delegates_to_the_component():
    translator = Translator(FakeTranslationModel())
    result = await translator.translate("Title", "Description.", ["a", "b"], "ENGLISH", "BENGALI")
    assert result["title"] == "BENGALI:Title"
    assert result["description"] == "BENGALI:Description."
    assert result["highlights"] == ["BENGALI:a", "BENGALI:b"]
