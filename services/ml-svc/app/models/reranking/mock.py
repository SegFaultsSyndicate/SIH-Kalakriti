"""services/ml-svc/app/models/reranking/mock.py

A crude lexical overlap score, so the ordering is at least explicable when a
human looks at a mock search result -- no weights loaded.
"""

from __future__ import annotations

import hashlib
import random


def _rng(*parts: object) -> random.Random:
    digest = hashlib.sha256("|".join(str(p) for p in parts).encode()).digest()
    return random.Random(int.from_bytes(digest[:8], "big"))


class MockTextReranker:
    def score(self, query: str, texts: list[str]) -> list[float]:
        terms = set(query.lower().split())
        scores = []
        for text in texts:
            words = set(text.lower().split())
            overlap = len(terms & words) / (len(terms) or 1)
            scores.append(round(overlap * 0.7 + _rng("rerank", query, text).random() * 0.3, 4))
        return scores
