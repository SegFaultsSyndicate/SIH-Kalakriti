"""services/ml-svc/app/models/embedding/mock.py

Deterministic unit vectors: same text, same vector, no weights loaded.
"""

from __future__ import annotations

import hashlib
import math
import random


def _rng(*parts: object) -> random.Random:
    digest = hashlib.sha256("|".join(str(p) for p in parts).encode()).digest()
    return random.Random(int.from_bytes(digest[:8], "big"))


def _unit_vector(text: str, dims: int = 768) -> list[float]:
    """A stable, L2-normalised vector for a string. Same text, same vector."""
    rng = _rng("embed", text)
    values = [rng.gauss(0, 1) for _ in range(dims)]
    norm = math.sqrt(sum(v * v for v in values)) or 1.0
    return [v / norm for v in values]


class MockTextEmbedder:
    def encode(self, texts: list[str]) -> list[list[float]]:
        return [_unit_vector(t) for t in texts]
