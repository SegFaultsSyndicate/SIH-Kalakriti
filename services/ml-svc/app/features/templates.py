"""services/ml-svc/app/features/templates.py

Pure, model-agnostic helpers shared by `extract_attributes.py`,
`verify_technique.py`, and `generate_description.py`. None of this touches a
model or protobuf, so all of it is trivially unit-testable with nothing
installed beyond the standard library -- and, deliberately, all of it runs
identically whether the component behind it was real or mock: a real
component's hallucination and a mock component's fabrication both flow
through the exact same guard.
"""

from __future__ import annotations

import logging
import re

log = logging.getLogger(__name__)


def _confidence(source: dict, key: str) -> float:
    try:
        return max(0.0, min(1.0, float(source.get(key, 0.0))))
    except (TypeError, ValueError):
        return 0.0


def _strings(value) -> list[str]:
    if not isinstance(value, list):
        return []
    return [str(v) for v in value if str(v).strip()]


def _vocab_hint(vocab: dict) -> str:
    """A prompt fragment naming a known craft's material/technique vocabulary.

    Only ever a hint, never the enforcement -- `_closed_vocab` below is what
    actually guarantees the answer stays on-list. Empty when the craft isn't
    declared yet or the seed CSV has no vocabulary columns for it.
    """
    materials = vocab.get("materials") or []
    techniques = vocab.get("techniques") or []
    if not materials and not techniques:
        return ""
    lines = []
    if materials:
        lines.append(f"materials known for this craft: {', '.join(materials)}")
    if techniques:
        lines.append(f"techniques known for this craft: {', '.join(techniques)}")
    return "\n" + "\n".join(lines)


def _closed_vocab(value: str, vocab: list[str], craft: str, field: str) -> str:
    """Keeps `value` only if it matches an entry of `vocab`, case-insensitively.

    A craft with no recorded vocabulary (`vocab == []`) means the seed CSV had
    no techniques/materials columns for it, not that anything goes -- absent
    vocabulary data isn't a constraint, the same rule `load_craft_allowlist`
    already follows for a plain-code-list CSV. So an empty vocab passes
    `value` through unchanged rather than rejecting it.
    """
    if not value or not vocab:
        return value
    by_lower = {v.lower(): v for v in vocab}
    match = by_lower.get(value.lower())
    if match is None:
        log.warning(
            "vlm proposed a %s outside the craft's vocabulary",
            field,
            extra={"craft_id": craft, field: value},
        )
        return ""
    return match


def _same_technique(observed: str, claimed: str) -> bool:
    normalise = lambda s: set(re.split(r"[\s\-_]+", s.lower().strip()))  # noqa: E731
    return bool(normalise(observed) & normalise(claimed))


def _template_sentence(facts: dict, craft_id: str) -> str:
    """A deterministic, factual sentence -- the description's grounded core.

    One clause per present attribute: nothing here is asserted unless `facts`
    actually carries it.
    """
    craft_name = craft_id.replace("-", " ").title() if craft_id else "handcrafted piece"
    clause = f"A {craft_name}"
    if facts.get("material"):
        clause += f" made of {facts['material']}"
    if facts.get("technique"):
        clause += f" using {facts['technique'].replace('-', ' ')}"
    sentences = [clause + "."]

    colours = facts.get("colours") or []
    if colours:
        sentences.append(f"Finished in {' and '.join(colours)}.")
    motifs = facts.get("motifs") or []
    if motifs:
        sentences.append(f"Carries {' and '.join(motifs)} motifs.")
    sentences.append("Each piece is finished by hand, so no two are identical.")
    return " ".join(sentences)


def _template_title(facts: dict, craft_id: str) -> str:
    craft_name = craft_id.replace("-", " ").title() if craft_id else "Handcrafted Piece"
    colour = next(iter(facts.get("colours") or []), None)
    parts = [p.title() for p in (colour, facts.get("material")) if p]
    parts.append(craft_name)
    return " ".join(parts)


def _template_highlights(facts: dict) -> list[str]:
    highlights = ["Hand-finished by a single artisan"]
    if facts.get("material"):
        highlights.append(f"Made from {facts['material']}")
    if facts.get("technique"):
        highlights.append(f"Crafted using {facts['technique'].replace('-', ' ')}")
    return highlights


def _template_keywords(facts: dict, craft_id: str) -> list[str]:
    keywords = [craft_id] if craft_id else []
    keywords += [facts[k] for k in ("material", "technique") if facts.get(k)]
    keywords += facts.get("colours") or []
    keywords += facts.get("motifs") or []
    return keywords


def _is_grounded(rewrite: str, facts: dict) -> bool:
    """True when every cited fact value still appears in `rewrite`,
    case-insensitively. Confirms values aren't dropped or swapped, not that no
    *other*, unrelated claim was invented around them -- good enough to catch
    a rewrite that drops or substitutes a material/colour; the artisan's
    review-before-publish step is the actual safety net for anything this
    substring check can't see.
    """
    if len(rewrite) < 10:
        return False
    lowered = rewrite.lower()
    for value in facts.values():
        for v in (value if isinstance(value, list) else [value]):
            if isinstance(v, str) and v and v.lower() not in lowered:
                return False
    return True
