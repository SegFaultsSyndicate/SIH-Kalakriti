"""services/ml-svc/app/models/mock.py

Deterministic stand-ins with the right shapes and plausible values, so the Go
side can build against a real wire contract with no weights on disk. Every
output is seeded from its input: the same request always gives the same answer,
different requests give different ones, and nothing is random between restarts.
"""

from __future__ import annotations

import asyncio
import hashlib
import math
import random

from app.config import Config, load_craft_allowlist

_MATERIALS = ["cotton", "muga silk", "tussar silk", "pashmina wool", "terracotta clay", "brass"]
_TECHNIQUES = [
    "hand-block-printing", "resist-dyeing", "pit-loom-weaving",
    "lost-wax-casting", "wheel-throwing", "tie-resist-dyeing",
]
_COLOURS = ["indigo", "madder red", "turmeric yellow", "natural ecru", "lac red", "cobalt blue"]
_MOTIFS = ["kalka", "jaapi", "buti", "jaal", "kalamkari border", "trellis"]
_HIGHLIGHTS = [
    "Hand-finished by a single artisan",
    "Natural dyes throughout",
    "Made to order in the maker's own cluster",
]


def _rng(*parts: object) -> random.Random:
    """A generator seeded by the request, so mock output is stable per input."""
    digest = hashlib.sha256("|".join(str(p) for p in parts).encode()).digest()
    return random.Random(int.from_bytes(digest[:8], "big"))


def _pick(rng: random.Random, options: list[str]) -> str:
    return options[rng.randrange(len(options))] if options else ""


class MockModels:
    """Every RPC, faked. No weights, no network, no torch."""

    def __init__(self, cfg: Config) -> None:
        self._cfg = cfg
        self.version = cfg.model_version
        # The same allowlist the real extractor is constrained to, so a mock
        # craft_id is a craft that actually exists in the ontology.
        self._crafts = load_craft_allowlist(cfg.craft_allowlist_path)

    async def load(self) -> None:
        """Nothing to load. Present so startup treats both backends alike."""
        return None

    async def enhance_image(
        self, object_key: str, remove_background: bool, auto_white_balance: bool, upscale: int
    ) -> tuple[str, list[str]]:
        ops = ["denoise"]
        if auto_white_balance:
            ops.append("auto-white-balance")
        if remove_background:
            ops.append("remove-background")
        if upscale > 1:
            ops.append(f"upscale-{upscale}x")
        return f"enhanced/{object_key.rsplit('/', 1)[-1].rsplit('.', 1)[0]}.webp", ops

    async def extract_attributes(
        self, object_keys: list[str], declared_craft_id: str, hint: str
    ) -> dict:
        rng = _rng("attrs", *object_keys, declared_craft_id, hint)
        craft = declared_craft_id if declared_craft_id in self._crafts else _pick(rng, self._crafts)
        return {
            "craft_id": (craft, 0.62 + rng.random() * 0.35 if craft else 0.0),
            "material": (_pick(rng, _MATERIALS), 0.55 + rng.random() * 0.4),
            "technique": (_pick(rng, _TECHNIQUES), 0.5 + rng.random() * 0.45),
            "colours": (rng.sample(_COLOURS, 2), 0.6 + rng.random() * 0.35),
            "motifs": (rng.sample(_MOTIFS, 2), 0.45 + rng.random() * 0.4),
            "dimensions": (
                {
                    "length_mm": rng.randrange(400, 2400, 50),
                    "width_mm": rng.randrange(300, 1200, 50),
                    "weight_g": rng.randrange(120, 1800, 10),
                },
                0.35 + rng.random() * 0.3,
            ),
        }

    async def generate_description(
        self, attributes: dict, craft_id: str, language: str, artisan_note: str, max_chars: int
    ) -> dict:
        rng = _rng("copy", craft_id, language, artisan_note)
        material = _value(attributes, "material") or "handspun cotton"
        technique = _value(attributes, "technique") or "hand-block-printing"
        colours = _values(attributes, "colours") or ["indigo"]
        motifs = _values(attributes, "motifs") or ["buti"]
        # Only keys that actually carried a value are claimed as sources, which is
        # the same rule the real generator is held to.
        used = [k for k in ("craft_id", "material", "technique", "colours", "motifs") if _has(attributes, k)]

        title = f"{colours[0].title()} {material.title()} {craft_id.replace('-', ' ').title()}"
        body = (
            f"Made in {technique.replace('-', ' ')} on {material}, "
            f"in {' and '.join(colours)}, carrying {' and '.join(motifs)} motifs. "
            f"Each piece is finished by hand, so no two are identical."
        )
        if artisan_note:
            body += f" In the maker's words: {artisan_note.strip()}"
        if max_chars > 0:
            body = body[:max_chars]

        return {
            "title": title[:120],
            "description": body,
            "highlights": rng.sample(_HIGHLIGHTS, 2),
            "keywords": [craft_id, material, technique, *colours],
            "attribute_keys_used": used,
            "language": language,
        }

    async def verify_technique(self, object_keys: list[str], claimed: str, craft_id: str) -> dict:
        rng = _rng("technique", *object_keys, claimed, craft_id)
        # Mostly agrees, so the happy path is the common path for the Go tests,
        # but deterministically disagrees for some inputs so the unhappy path is
        # reachable without a real model.
        matches = rng.random() > 0.2
        observed = claimed if matches else _pick(rng, _TECHNIQUES)
        return {
            "claimed": claimed,
            "observed": observed,
            "matches": matches,
            "confidence": 0.7 + rng.random() * 0.29 if matches else 0.4 + rng.random() * 0.3,
            "explanation": (
                f"The weave and edge finish are consistent with {observed.replace('-', ' ')}."
                if matches
                else f"The surface reads as {observed.replace('-', ' ')} rather than {claimed.replace('-', ' ')}."
            ),
        }

    async def detect_handloom(self, object_key: str, declared_thread_count: int) -> dict:
        rng = _rng("loom", object_key, declared_thread_count)
        # Powerloom weave is far more periodic, so a high peak-to-median ratio is
        # the machine-made case; the mock keeps that relationship intact.
        ratio = 1.6 + rng.random() * 6.0
        is_handloom = ratio < 4.0
        return {
            "is_handloom": is_handloom,
            "confidence": 0.6 + rng.random() * 0.35,
            "fft_peak_ratio": ratio,
            "explanation": (
                f"Thread spacing varies across the crop (peak-to-median {ratio:.1f}), which is "
                "what a hand-thrown shuttle looks like."
                if is_handloom
                else f"Thread spacing is highly regular (peak-to-median {ratio:.1f}), which points to a powerloom."
            ),
        }

    async def embed(self, texts: list[str]) -> list[list[float]]:
        return [_unit_vector(t) for t in texts]

    async def rerank(self, query: str, candidates: list[tuple[str, str]]) -> list[tuple[str, float]]:
        # A crude lexical overlap, so the ordering is at least explicable when a
        # human looks at a mock search result.
        terms = set(query.lower().split())
        scored = []
        for cid, text in candidates:
            words = set(text.lower().split())
            overlap = len(terms & words) / (len(terms) or 1)
            scored.append((cid, round(overlap * 0.7 + _rng("rerank", query, cid).random() * 0.3, 4)))
        return sorted(scored, key=lambda p: p[1], reverse=True)

    async def transcribe(self, object_key: str, language: str, interim: bool):
        """Streams the same chunk shapes Bhashini does, in the same order."""
        rng = _rng("asr", object_key, language)
        sentences = [
            "मेरा नाम है और मैं यह काम बीस साल से कर रहा हूँ।",
            "यह कपड़ा हाथ से रंगा गया है।",
            "इसमें प्राकृतिक रंग इस्तेमाल हुए हैं।",
        ]
        offset = 0
        for sentence in sentences[: 1 + rng.randrange(len(sentences))]:
            duration = 900 + rng.randrange(1200)
            if interim:
                half = len(sentence) // 2
                yield {
                    "text": sentence[:half],
                    "is_final": False,
                    "start_ms": offset,
                    "end_ms": offset + duration // 2,
                    "confidence": 0.0,
                    "language": language,
                }
                await asyncio.sleep(0)
            yield {
                "text": sentence,
                "is_final": True,
                "start_ms": offset,
                "end_ms": offset + duration,
                "confidence": 0.82 + rng.random() * 0.15,
                "language": language,
            }
            offset += duration


# --- helpers ----------------------------------------------------------------


def _unit_vector(text: str, dims: int = 768) -> list[float]:
    """A stable, L2-normalised vector for a string. Same text, same vector."""
    rng = _rng("embed", text)
    values = [rng.gauss(0, 1) for _ in range(dims)]
    norm = math.sqrt(sum(v * v for v in values)) or 1.0
    return [v / norm for v in values]


def _has(attributes: dict, key: str) -> bool:
    value = attributes.get(key)
    if isinstance(value, tuple):
        value = value[0]
    return bool(value)


def _value(attributes: dict, key: str) -> str:
    value = attributes.get(key)
    return value[0] if isinstance(value, tuple) else (value or "")


def _values(attributes: dict, key: str) -> list[str]:
    value = attributes.get(key)
    value = value[0] if isinstance(value, tuple) else value
    return list(value or [])
