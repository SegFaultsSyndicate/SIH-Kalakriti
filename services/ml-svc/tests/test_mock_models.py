"""services/ml-svc/tests/test_mock_models.py

The mock backend, checked without protobuf: shapes, determinism, and the two
rules the Go side depends on (768-dim unit vectors, craft ids from the ontology).
"""

import math

from app.config import Config, load_craft_allowlist
from app.models import build


def _registry():
    """A fresh mock registry. A plain call, not a fixture: these tests run under
    anything that can await a coroutine."""
    return build(Config(mock_mode=True))


async def test_starts_with_no_weights():
    registry = _registry()
    await registry.load()
    assert registry.version


async def test_embeddings_are_768_dim_unit_vectors_and_stable():
    registry = _registry()
    vectors = await registry.embed(["ajrakh dupatta", "अजरख दुपट्टा"])

    for v in vectors:
        assert len(v) == 768
        assert math.isclose(math.sqrt(sum(x * x for x in v)), 1.0, rel_tol=1e-9)

    # Deterministic: the same text always gives the same vector, or a cached
    # pgvector row would drift under the Go side's feet.
    assert (await registry.embed(["ajrakh dupatta"]))[0] == vectors[0]
    assert vectors[0] != vectors[1]


async def test_extract_never_invents_a_craft_id():
    registry = _registry()
    allowlist = set(load_craft_allowlist(Config().craft_allowlist_path))
    assert allowlist, "the seed CSV should be readable from the service"

    honoured = await registry.extract_attributes(["a.jpg"], "ajrakh-block-printing", "")
    assert honoured["craft_id"][0] == "ajrakh-block-printing"

    invented = await registry.extract_attributes(["a.jpg"], "definitely-not-a-craft", "")
    assert invented["craft_id"][0] in allowlist


async def test_extract_returns_a_confidence_per_field():
    registry = _registry()
    attributes = await registry.extract_attributes(["a.jpg", "b.jpg"], "", "hand printed")

    for key in ("craft_id", "material", "technique", "colours", "motifs", "dimensions"):
        _, confidence = attributes[key]
        assert 0.0 <= confidence <= 1.0, key


async def test_description_only_cites_attributes_it_was_given():
    registry = _registry()
    attributes = await registry.extract_attributes(["a.jpg"], "blue-pottery", "")
    copy = await registry.generate_description(attributes, "blue-pottery", "ENGLISH", "my story", 200)

    assert copy["title"] and copy["description"]
    assert len(copy["description"]) <= 200
    assert set(copy["attribute_keys_used"]) <= set(attributes)


async def test_handloom_verdict_agrees_with_its_own_ratio():
    registry = _registry()
    for key in ("a.jpg", "b.jpg", "c.jpg", "d.jpg"):
        verdict = await registry.detect_handloom(key, 120)
        # Regular spacing means powerloom; the explanation must not contradict
        # the number it quotes.
        assert verdict["is_handloom"] == (verdict["fft_peak_ratio"] < 4.0)
        assert f"{verdict['fft_peak_ratio']:.1f}" in verdict["explanation"]


async def test_rerank_returns_every_candidate_best_first():
    registry = _registry()
    candidates = [("a", "ajrakh cotton stole"), ("b", "brass lamp"), ("c", "ajrakh indigo dupatta")]
    results = await registry.rerank("ajrakh stole", candidates)

    assert {cid for cid, _ in results} == {"a", "b", "c"}
    assert [s for _, s in results] == sorted((s for _, s in results), reverse=True)


async def test_transcribe_streams_final_chunks_in_order():
    registry = _registry()
    chunks = [c async for c in registry.transcribe("note.m4a", "HINDI", True)]

    assert chunks
    assert any(c["is_final"] for c in chunks)
    finals = [c for c in chunks if c["is_final"]]
    assert finals == sorted(finals, key=lambda c: c["start_ms"])
    for chunk in finals:
        assert chunk["end_ms"] > chunk["start_ms"]
        assert 0.0 <= chunk["confidence"] <= 1.0


async def test_enhance_names_the_operations_it_applied():
    registry = _registry()
    key, ops = await registry.enhance_image("artisans/x/y.jpg", True, True, 2)

    assert key.startswith("enhanced/") and key.endswith(".webp")
    assert ops == ["denoise", "auto-white-balance", "remove-background", "upscale-2x"]


async def test_verify_technique_echoes_the_claim():
    registry = _registry()
    verdict = await registry.verify_technique(["clip.mp4"], "pit-loom-weaving", "assam-muga-weaving")

    assert verdict["claimed"] == "pit-loom-weaving"
    assert verdict["observed"]
    assert verdict["matches"] == (verdict["observed"] == "pit-loom-weaving")
    assert verdict["explanation"]
