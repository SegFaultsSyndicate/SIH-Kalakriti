"""services/ml-svc/app/models/vlm/mock.py

Deterministic stand-ins with the right shapes and plausible values, so the Go
side can build against a real wire contract with no weights on disk. Every
output is seeded from its input: the same request always gives the same
answer, different requests give different ones.

`propose_attributes` draws material/technique from the given craft's own
seeded vocabulary when one is recorded, falling back to a small generic list
otherwise -- the same "absent vocabulary isn't a constraint" rule
`features/templates.py`'s `_closed_vocab` follows, so the feature's
enforcement pass this component's raw guess flows through afterwards is
normally a no-op for mock, exactly as it should be: mock has nothing to guard
against.
"""

from __future__ import annotations

import hashlib
import random

_MATERIALS = ["cotton", "muga silk", "tussar silk", "pashmina wool", "terracotta clay", "brass"]
_TECHNIQUES = [
    "hand-block-printing", "resist-dyeing", "pit-loom-weaving",
    "lost-wax-casting", "wheel-throwing", "tie-resist-dyeing",
]
_COLOURS = ["indigo", "madder red", "turmeric yellow", "natural ecru", "lac red", "cobalt blue"]
_MOTIFS = ["kalka", "jaapi", "buti", "jaal", "kalamkari border", "trellis"]


def _rng(*parts: object) -> random.Random:
    digest = hashlib.sha256("|".join(str(p) for p in parts).encode()).digest()
    return random.Random(int.from_bytes(digest[:8], "big"))


def _pick(rng: random.Random, options: list[str]) -> str:
    return options[rng.randrange(len(options))] if options else ""


class MockVisionLanguageModel:
    def propose_attributes(
        self,
        images: list[bytes],
        object_keys: list[str],
        allowlist: list[str],
        declared_craft_id: str,
        hint: str,
        vocab_hint: str,
        craft_vocab: dict[str, dict[str, list[str]]],
    ) -> dict:
        rng = _rng("attrs", *object_keys, declared_craft_id, hint)
        craft = declared_craft_id if declared_craft_id in allowlist else _pick(rng, allowlist)
        vocab = craft_vocab.get(craft, {})
        material = _pick(rng, vocab.get("materials") or _MATERIALS)
        technique = _pick(rng, vocab.get("techniques") or _TECHNIQUES)
        return {
            "craft_id": craft,
            "material": material,
            "technique": technique,
            "colours": rng.sample(_COLOURS, 2),
            "motifs": rng.sample(_MOTIFS, 2),
            "confidence": {
                "craft_id": 0.62 + rng.random() * 0.35 if craft else 0.0,
                "material": 0.55 + rng.random() * 0.4,
                "technique": 0.5 + rng.random() * 0.45,
                "colours": 0.6 + rng.random() * 0.35,
                "motifs": 0.45 + rng.random() * 0.4,
            },
            "dimensions": {
                "length_mm": rng.randrange(400, 2400, 50),
                "width_mm": rng.randrange(300, 1200, 50),
                "weight_g": rng.randrange(120, 1800, 10),
            },
            "dimensions_confidence": 0.35 + rng.random() * 0.3,
        }

    def observe_technique(
        self,
        media: list[bytes],
        object_keys: list[str],
        claimed: str,
        craft_id: str,
        video_frames: int,
    ) -> dict:
        rng = _rng("technique", *object_keys, claimed, craft_id)
        # Mostly agrees, so the happy path is the common path for the Go tests,
        # but deterministically disagrees for some inputs so the unhappy path is
        # reachable without a real model.
        matches = rng.random() > 0.2
        observed = claimed if matches else _pick(rng, _TECHNIQUES)
        return {
            "observed": observed,
            "matches": matches,
            "confidence": 0.7 + rng.random() * 0.29 if matches else 0.4 + rng.random() * 0.3,
            "explanation": (
                f"The weave and edge finish are consistent with {observed.replace('-', ' ')}."
                if matches
                else f"The surface reads as {observed.replace('-', ' ')} rather than {claimed.replace('-', ' ')}."
            ),
        }

    def polish(self, template: str, facts: dict, language: str, max_chars: int) -> str:
        # A no-op passthrough: grounded by construction, since nothing about
        # `template` changes. Real mode is where an actual rewrite -- and the
        # grounding check on it -- happens; see `features/generate_description.py`.
        return template
