"""services/ml-svc/tests/models/test_vlm_llamacpp.py

Needs httpx (the `real` extra, not installed by the default `dev`-only CI run
this test suite otherwise targets), so this `importorskip`s rather than
assumes it is there -- same convention as `test_image_lighting_net.py`. No
real `llama-server` is needed: `httpx.post` is monkeypatched, so this checks
the request shape and response parsing, not a live model.
"""

import json

import pytest

from app.models.vlm import VLMConfig


def _fake_response(monkeypatch, payload: dict, capture: dict):
    httpx = pytest.importorskip("httpx")

    class FakeResponse:
        def raise_for_status(self):
            pass

        def json(self):
            return {"choices": [{"message": {"content": json.dumps(payload)}}]}

    def fake_post(url, json=None, timeout=None):  # noqa: A002
        capture["url"] = url
        capture["body"] = json
        return FakeResponse()

    monkeypatch.setattr(httpx, "post", fake_post)


def test_propose_attributes_sends_images_and_json_schema(monkeypatch):
    from app.models.vlm.llamacpp import LlamaCppVisionLanguageModel

    capture: dict = {}
    _fake_response(
        monkeypatch,
        {
            "craft_id": "ajrakh-block-printing",
            "material": "cotton",
            "technique": "hand-block-printing",
            "colours": ["indigo"],
            "motifs": ["buti"],
            "confidence": {"craft_id": 0.9, "material": 0.8, "technique": 0.7, "colours": 0.6, "motifs": 0.5},
        },
        capture,
    )

    backend = LlamaCppVisionLanguageModel(VLMConfig(llamacpp_base_url="http://llama-server:8080"))
    result = backend.propose_attributes(
        [b"fake-jpeg-bytes"], ["a.jpg"], ["ajrakh-block-printing"], "ajrakh-block-printing", "", "", {}
    )

    assert result["craft_id"] == "ajrakh-block-printing"
    assert result["material"] == "cotton"
    assert capture["url"] == "http://llama-server:8080/v1/chat/completions"
    content = capture["body"]["messages"][0]["content"]
    assert content[0]["type"] == "text"
    assert content[1]["type"] == "image_url"
    assert content[1]["image_url"]["url"].startswith("data:image/jpeg;base64,")
    assert capture["body"]["response_format"]["type"] == "json_schema"
    assert capture["body"]["temperature"] == 0


def test_polish_sends_no_images(monkeypatch):
    from app.models.vlm.llamacpp import LlamaCppVisionLanguageModel

    httpx = pytest.importorskip("httpx")

    class FakeResponse:
        def raise_for_status(self):
            pass

        def json(self):
            return {"choices": [{"message": {"content": "A warmer rewrite."}}]}

    captured = {}

    def fake_post(url, json=None, timeout=None):  # noqa: A002
        captured["body"] = json
        return FakeResponse()

    monkeypatch.setattr(httpx, "post", fake_post)

    backend = LlamaCppVisionLanguageModel(VLMConfig())
    result = backend.polish("A stole made of cotton.", {}, "ENGLISH", 900)

    assert result == "A warmer rewrite."
    content = captured["body"]["messages"][0]["content"]
    assert len(content) == 1
    assert content[0]["type"] == "text"
    assert "response_format" not in captured["body"]


def test_propose_attributes_handles_unparseable_reply_by_abstaining(monkeypatch):
    from app.models.vlm.llamacpp import LlamaCppVisionLanguageModel

    httpx = pytest.importorskip("httpx")

    class FakeResponse:
        def raise_for_status(self):
            pass

        def json(self):
            return {"choices": [{"message": {"content": "not json at all"}}]}

    monkeypatch.setattr(httpx, "post", lambda *a, **k: FakeResponse())

    backend = LlamaCppVisionLanguageModel(VLMConfig())
    result = backend.propose_attributes([b"x"], ["a.jpg"], ["craft"], "", "", "", {})

    assert result["craft_id"] == ""
    assert result["confidence"] == {}
