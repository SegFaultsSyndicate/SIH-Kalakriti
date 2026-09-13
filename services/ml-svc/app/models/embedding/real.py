"""services/ml-svc/app/models/embedding/real.py

sentence-transformers-backed embedding. Imported only when real mode selects
this component, so mock mode never needs torch/sentence-transformers on disk.
"""

from __future__ import annotations

from app.model_loading import local_files_only_kwargs, resolve_device
from app.models.embedding import EmbeddingConfig


class RealTextEmbedder:
    def __init__(self, cfg: EmbeddingConfig) -> None:
        from sentence_transformers import SentenceTransformer

        device, _ = resolve_device()
        # local_files_only_kwargs avoids a Hub network round-trip on every
        # restart once this repo id is actually cached: it is pinned, not
        # floating, so there is never a newer revision to check for.
        self._model = SentenceTransformer(cfg.model, device=device, **local_files_only_kwargs(cfg.model))

    def encode(self, texts: list[str]) -> list[list[float]]:
        # e5 wants the prefix; without it retrieval quality drops noticeably.
        prefixed = [t if t.startswith(("query:", "passage:")) else f"passage: {t}" for t in texts]
        vectors = self._model.encode(prefixed, normalize_embeddings=True, batch_size=len(prefixed))
        return [v.tolist() for v in vectors]
