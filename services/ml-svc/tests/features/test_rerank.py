"""services/ml-svc/tests/features/test_rerank.py

Uses a hand-written fake, not the real component's mock -- exercising this
feature does not need app.models.reranking at all.
"""

from app.features.rerank import Reranker


class FakeReranker:
    def __init__(self, scores: dict[str, float]) -> None:
        self._scores = scores

    def score(self, query: str, texts: list[str]) -> list[float]:
        return [self._scores[t] for t in texts]


async def test_rerank_pairs_ids_and_sorts_best_first():
    reranker = Reranker(FakeReranker({"a": 0.2, "b": 0.9}))
    result = await reranker.rerank("q", [("id-a", "a"), ("id-b", "b")])
    assert result == [("id-b", 0.9), ("id-a", 0.2)]


async def test_rerank_empty_candidates_short_circuits():
    reranker = Reranker(FakeReranker({}))
    assert await reranker.rerank("q", []) == []
