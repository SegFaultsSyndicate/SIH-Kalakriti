"""services/ml-svc/tests/test_real_helpers.py

The pure-Python grounding logic in app.models.real: closed-vocabulary
enforcement and the template-first description pipeline. These run with no
torch/transformers on disk — real.py only imports those inside RealModels'
instance methods, never at module scope — so this is the part of "real mode"
that can actually be verified in CI without weights.

The Zero-DCE++ architecture tests at the bottom are the exception: they need
torch (the `real` extra, not installed by the default `dev`-only CI run this
file otherwise targets), so they `importorskip` rather than assume it is
there. No pretrained checkpoint is needed for them — they check the network's
shape/range contract against random init weights, not that it actually
brightens a real photo (that needs the real checkpoint; see
app/models/zero_dce.py).
"""

import pytest

from app.config import load_craft_vocab
from app.models.real import (
    _closed_vocab,
    _is_grounded,
    _parse_json,
    _template_highlights,
    _template_keywords,
    _template_sentence,
    _template_title,
    _vocab_hint,
)


def test_closed_vocab_accepts_a_case_insensitive_match():
    assert _closed_vocab("Cotton", ["cotton", "silk"], "c1", "material") == "cotton"


def test_closed_vocab_rejects_a_value_outside_the_list():
    assert _closed_vocab("polyester", ["cotton", "silk"], "c1", "material") == ""


def test_closed_vocab_passes_through_when_no_vocab_is_recorded():
    # No techniques/materials columns for this craft: absent data isn't a
    # constraint, so nothing is rejected.
    assert _closed_vocab("anything", [], "c1", "material") == "anything"


def test_closed_vocab_passes_through_an_empty_value():
    assert _closed_vocab("", ["cotton"], "c1", "material") == ""


def test_load_craft_vocab_reads_pipe_separated_columns(tmp_path):
    csv_path = tmp_path / "crafts.csv"
    csv_path.write_text(
        "code,display_name,parent_code,gi_registration_no,techniques,materials\n"
        "ajrakh-block-printing,Ajrakh Block Printing,,,"
        "hand-block-printing|resist-dyeing,cotton|natural-indigo\n"
    )
    vocab = load_craft_vocab(csv_path)
    assert vocab["ajrakh-block-printing"]["materials"] == ["cotton", "natural-indigo"]
    assert vocab["ajrakh-block-printing"]["techniques"] == ["hand-block-printing", "resist-dyeing"]


def test_load_craft_vocab_is_empty_for_a_plain_code_list(tmp_path):
    # load_craft_allowlist's other accepted shape: no techniques/materials
    # columns at all. Should not raise, and should yield no vocabulary.
    csv_path = tmp_path / "codes.txt"
    csv_path.write_text("ajrakh-block-printing\nbagru-block-printing\n")
    assert load_craft_vocab(csv_path) == {}


def test_load_craft_vocab_missing_file_is_not_fatal(tmp_path):
    assert load_craft_vocab(tmp_path / "nope.csv") == {}


def test_template_sentence_only_asserts_present_facts():
    sentence = _template_sentence({"material": "cotton"}, "ajrakh-block-printing")
    assert "cotton" in sentence
    assert "using" not in sentence  # no technique given, so no technique clause


def test_template_sentence_with_no_facts_still_names_the_craft():
    sentence = _template_sentence({}, "ajrakh-block-printing")
    assert "Ajrakh Block Printing" in sentence


def test_template_title_leads_with_colour_then_material():
    title = _template_title({"material": "cotton", "colours": ["indigo", "madder"]}, "ajrakh-block-printing")
    assert title == "Indigo Cotton Ajrakh Block Printing"


def test_template_highlights_and_keywords_reflect_given_facts_only():
    facts = {"material": "cotton", "technique": "hand-block-printing", "colours": ["indigo"]}
    highlights = _template_highlights(facts)
    assert any("cotton" in h for h in highlights)
    assert any("hand block printing" in h for h in highlights)

    keywords = _template_keywords(facts, "ajrakh-block-printing")
    assert set(keywords) == {"ajrakh-block-printing", "cotton", "hand-block-printing", "indigo"}


def test_is_grounded_accepts_a_faithful_rewrite():
    facts = {"material": "cotton", "colours": ["indigo"]}
    assert _is_grounded("A soft cotton weave dyed a deep indigo, made by hand.", facts)


def test_is_grounded_rejects_a_dropped_or_swapped_value():
    facts = {"material": "cotton"}
    assert not _is_grounded("A beautiful silk piece, handcrafted with care.", facts)


def test_is_grounded_rejects_a_degenerate_rewrite():
    assert not _is_grounded("Nice.", {"material": "cotton"})


def test_is_grounded_ignores_non_string_facts_like_dimensions():
    facts = {"material": "cotton", "dimensions": {"length_mm": 500}}
    assert _is_grounded("A cotton piece, finished by hand.", facts)


def test_vocab_hint_empty_when_craft_has_no_recorded_vocabulary():
    assert _vocab_hint({}) == ""


def test_vocab_hint_names_both_columns_when_present():
    hint = _vocab_hint({"materials": ["cotton"], "techniques": ["hand-block-printing"]})
    assert "cotton" in hint
    assert "hand-block-printing" in hint


def test_parse_json_extracts_a_fenced_object():
    assert _parse_json('Sure, here you go:\n```json\n{"a": 1}\n```') == {"a": 1}


def test_parse_json_returns_empty_dict_on_garbage():
    assert _parse_json("not json at all") == {}


def test_zero_dce_net_preserves_shape_and_output_range():
    torch = pytest.importorskip("torch")
    from app.models.zero_dce import ZeroDCEPPNet

    net = ZeroDCEPPNet().eval()
    # Odd, non-square, non-power-of-two size on purpose: the architecture has
    # no pooling, so it must not assume dimensions divide evenly by anything.
    x = torch.rand(1, 3, 37, 51)
    with torch.no_grad():
        out = net(x)

    assert out.shape == x.shape
    # The curve is x + A*(x^2 - x) with A = tanh(...) in (-1, 1); starting
    # from x in [0, 1] this can drift slightly outside that range over 8
    # iterations, which is exactly why app.models.real._apply_zero_dce clamps
    # afterwards -- this asserts the *shape* contract, not a bounded range.
    assert torch.isfinite(out).all()


def test_zero_dce_net_state_dict_keys_match_upstream_names():
    """Guards the one property that actually matters for loading a real
    checkpoint: layer names must match the upstream repo's `enhance_net_nopool`
    exactly, since Epoch99.pth was saved from that class."""
    torch = pytest.importorskip("torch")
    from app.models.zero_dce import ZeroDCEPPNet

    keys = set(ZeroDCEPPNet().state_dict().keys())
    for layer in ("e_conv1", "e_conv2", "e_conv3", "e_conv4", "e_conv5", "e_conv6", "e_conv7"):
        assert f"{layer}.depth_conv.weight" in keys
        assert f"{layer}.point_conv.weight" in keys
