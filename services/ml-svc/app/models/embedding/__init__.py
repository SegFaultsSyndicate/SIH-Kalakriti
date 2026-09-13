"""services/ml-svc/app/models/embedding/

The text-embedding component. Swappable independently of reranking, the VLM,
or anything else: `ML_SVC_TEXT_EMBEDDING_MODEL` names a SentenceTransformer
repo id today; a future backend (a hosted embeddings API, say) implements the
same `TextEmbedder` Protocol with no change to `features/embed.py`.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class TextEmbedder(Protocol):
    """Sync on purpose -- features wrap the call in `asyncio.to_thread`, same
    as every other CPU-bound component here."""

    def encode(self, texts: list[str]) -> list[list[float]]: ...


@dataclass(frozen=True)
class EmbeddingConfig:
    model: str = "intfloat/multilingual-e5-base"

    @classmethod
    def from_env(cls) -> "EmbeddingConfig":
        return cls(model=os.getenv("ML_SVC_TEXT_EMBEDDING_MODEL", "intfloat/multilingual-e5-base"))


def build(cfg: Config) -> TextEmbedder:
    if cfg.mock_mode:
        from app.models.embedding.mock import MockTextEmbedder

        return MockTextEmbedder()

    from app.models.embedding.real import RealTextEmbedder

    return RealTextEmbedder(EmbeddingConfig.from_env())
