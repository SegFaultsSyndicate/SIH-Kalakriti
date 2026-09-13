"""services/ml-svc/app/features/rerank.py

The Rerank RPC's business logic: id-pairing and best-first sorting are pure
and model-agnostic, so they live here rather than inside either backend of
the reranking component.
"""

from __future__ import annotations

import asyncio

from app.models.reranking import TextReranker


class Reranker:
    def __init__(self, reranking: TextReranker) -> None:
        self._reranking = reranking

    async def rerank(self, query: str, candidates: list[tuple[str, str]]) -> list[tuple[str, float]]:
        if not candidates:
            return []
        texts = [text for _, text in candidates]
        scores = await asyncio.to_thread(self._reranking.score, query, texts)
        pairs = [(cid, score) for (cid, _), score in zip(candidates, scores)]
        return sorted(pairs, key=lambda p: p[1], reverse=True)
