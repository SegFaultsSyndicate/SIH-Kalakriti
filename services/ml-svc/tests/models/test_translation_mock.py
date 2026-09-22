"""services/ml-svc/tests/models/test_translation_mock.py"""

from app.config import Config
from app.models import translation


def _backend():
    return translation.build(Config(mock_mode=True))


def test_translate_tags_every_field_with_the_target_language():
    backend = _backend()
    result = backend.translate("A cotton scarf.", "Hand woven.", ["soft", "durable"], "ENGLISH", "HINDI")
    assert result["title"] == "[HINDI] A cotton scarf."
    assert result["description"] == "[HINDI] Hand woven."
    assert result["highlights"] == ["[HINDI] soft", "[HINDI] durable"]


def test_translate_never_touches_a_do_not_translate_placeholder():
    backend = _backend()
    result = backend.translate("{{dnt0}} scarf.", "Made with {{dnt0}}.", [], "ENGLISH", "TAMIL")
    assert "{{dnt0}}" in result["title"]
    assert "{{dnt0}}" in result["description"]
