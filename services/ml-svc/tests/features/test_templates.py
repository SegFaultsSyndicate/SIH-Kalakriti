"""services/ml-svc/tests/features/test_templates.py

Pure functions shared by extract_attributes/verify_technique/
generate_description -- no model, no torch, nothing installed beyond stdlib.
"""

from app.features.templates import (
    _closed_vocab,
    _is_grounded,
    _same_technique,
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
    assert _closed_vocab("anything", [], "c1", "material") == "anything"


def test_closed_vocab_passes_through_an_empty_value():
    assert _closed_vocab("", ["cotton"], "c1", "material") == ""


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


def test_same_technique_matches_on_shared_words():
    assert _same_technique("hand block printing", "hand-block-printing")


def test_same_technique_rejects_unrelated_words():
    assert not _same_technique("wheel throwing", "resist dyeing")
