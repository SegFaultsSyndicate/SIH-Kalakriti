"""services/ml-svc/app/models/reranking/real.py

CrossEncoder-backed reranking. Imported only when real mode selects this
component, so mock mode never needs torch/sentence-transformers on disk.
"""

from __future__ import annotations

from app.model_loading import local_files_only_kwargs, resolve_device
from app.models.reranking import RerankingConfig


class RealTextReranker:
    def __init__(self, cfg: RerankingConfig) -> None:
        from sentence_transformers import CrossEncoder

        device = cfg.device or resolve_device()[0]
        self._model = CrossEncoder(cfg.model, device=device, **local_files_only_kwargs(cfg.model))

    def score(self, query: str, texts: list[str]) -> list[float]:
        if not texts:
            return []
        return [float(s) for s in self._model.predict([(query, text) for text in texts])]
