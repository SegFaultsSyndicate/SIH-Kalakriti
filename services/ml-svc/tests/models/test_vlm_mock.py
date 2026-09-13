"""services/ml-svc/tests/models/test_vlm_mock.py"""

from app.config import Config, load_craft_allowlist
from app.models import vlm


def _backend():
    return vlm.build(Config(mock_mode=True))


def test_propose_attributes_never_invents_a_craft_id():
    backend = _backend()
    allowlist = load_craft_allowlist(Config().craft_allowlist_path)
    assert allowlist, "the seed CSV should be readable from the service"

    honoured = backend.propose_attributes([b""], ["a.jpg"], allowlist, "ajrakh-block-printing", "", "", {})
    assert honoured["craft_id"] == "ajrakh-block-printing"

    invented = backend.propose_attributes([b""], ["a.jpg"], allowlist, "definitely-not-a-craft", "", "", {})
    assert invented["craft_id"] in allowlist


def test_propose_attributes_draws_from_the_crafts_own_vocabulary_when_given_one():
    backend = _backend()
    craft_vocab = {"ajrakh-block-printing": {"materials": ["cotton"], "techniques": ["hand-block-printing"]}}
    raw = backend.propose_attributes(
        [b""], ["a.jpg"], ["ajrakh-block-printing"], "ajrakh-block-printing", "", "", craft_vocab
    )
    assert raw["material"] == "cotton"
    assert raw["technique"] == "hand-block-printing"


def test_propose_attributes_returns_a_confidence_per_field():
    backend = _backend()
    raw = backend.propose_attributes([b"", b""], ["a.jpg", "b.jpg"], [], "", "hand printed", "", {})
    for key in ("craft_id", "material", "technique", "colours", "motifs"):
        assert 0.0 <= raw["confidence"][key] <= 1.0


def test_observe_technique_echoes_the_claim():
    backend = _backend()
    verdict = backend.observe_technique([b""], ["clip.mp4"], "pit-loom-weaving", "assam-muga-weaving", 8)
    assert verdict["observed"]
    assert verdict["matches"] == (verdict["observed"] == "pit-loom-weaving")
    assert verdict["explanation"]


def test_polish_is_a_grounded_by_construction_passthrough():
    backend = _backend()
    assert backend.polish("A cotton piece.", {"material": "cotton"}, "ENGLISH", 200) == "A cotton piece."
