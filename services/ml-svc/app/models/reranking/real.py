"""services/ml-svc/app/models/reranking/real.py

CrossEncoder-backed reranking. Imported only when real mode selects this
component, so mock mode never needs torch/sentence-transformers on disk.
"""

from __future__ import annotations

from app.model_loading import load_or_export_onnx, local_files_only_kwargs, resolve_device
from app.models.reranking import RerankingConfig


class RealTextReranker:
    def __init__(self, cfg: RerankingConfig) -> None:
        from sentence_transformers import CrossEncoder

        device = cfg.device or resolve_device()[0]

        def _load(model_name_or_path: str) -> CrossEncoder:
            # See RealTextEmbedder's comment -- same local_files_only scope.
            local_only = local_files_only_kwargs(cfg.model) if model_name_or_path == cfg.model else {}
            return CrossEncoder(model_name_or_path, device=device, backend=cfg.backend, **local_only)

        # See RealTextEmbedder's comment -- same persist-the-export reason.
        self._model = load_or_export_onnx(_load, cfg.model) if cfg.backend == "onnx" else _load(cfg.model)

    def score(self, query: str, texts: list[str]) -> list[float]:
        if not texts:
            return []
        return [float(s) for s in self._model.predict([(query, text) for text in texts])]
