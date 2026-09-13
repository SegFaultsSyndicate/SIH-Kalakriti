"""services/ml-svc/app/models/reranking/

The text-reranking component. Scores (query, text) pairs; id-pairing and
sorting are pure, model-agnostic logic and live in `features/rerank.py`
instead, so a future backend only has to implement `score`.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class TextReranker(Protocol):
    def score(self, query: str, texts: list[str]) -> list[float]: ...


@dataclass(frozen=True)
class RerankingConfig:
    model: str = "BAAI/bge-reranker-v2-m3"

    @classmethod
    def from_env(cls) -> "RerankingConfig":
        return cls(model=os.getenv("ML_SVC_TEXT_RERANKING_MODEL", "BAAI/bge-reranker-v2-m3"))


def build(cfg: Config) -> TextReranker:
    if cfg.mock_mode:
        from app.models.reranking.mock import MockTextReranker

        return MockTextReranker()

    from app.models.reranking.real import RealTextReranker

    return RealTextReranker(RerankingConfig.from_env())
