"""services/ml-svc/app/models/embedding/real.py

sentence-transformers-backed embedding. Imported only when real mode selects
this component, so mock mode never needs torch/sentence-transformers on disk.
"""

from __future__ import annotations

from app.model_loading import load_or_export_onnx, local_files_only_kwargs, resolve_device
from app.models.embedding import EmbeddingConfig


class RealTextEmbedder:
    def __init__(self, cfg: EmbeddingConfig) -> None:
        from sentence_transformers import SentenceTransformer

        device = cfg.device or resolve_device()[0]

        def _load(model_name_or_path: str) -> SentenceTransformer:
            # local_files_only_kwargs avoids a Hub network round-trip on
            # every restart once cfg.model is actually cached: it is
            # pinned, not floating, so there is never a newer revision to
            # check for. Only applies when loading cfg.model's repo id
            # directly -- a local onnx export dir (see load_or_export_onnx
            # below) is already fully local, no Hub lookup either way.
            local_only = local_files_only_kwargs(cfg.model) if model_name_or_path == cfg.model else {}
            return SentenceTransformer(model_name_or_path, device=device, backend=cfg.backend, **local_only)

        # sentence-transformers only holds an on-the-fly ONNX export (a repo
        # with no pre-published onnx/model.onnx) in memory, not on disk --
        # load_or_export_onnx persists it so a later restart skips the
        # export entirely. A no-op wrapper for backend="torch".
        self._model = load_or_export_onnx(_load, cfg.model) if cfg.backend == "onnx" else _load(cfg.model)

    def encode(self, texts: list[str]) -> list[list[float]]:
        # e5 wants the prefix; without it retrieval quality drops noticeably.
        prefixed = [t if t.startswith(("query:", "passage:")) else f"passage: {t}" for t in texts]
        vectors = self._model.encode(prefixed, normalize_embeddings=True, batch_size=len(prefixed))
        return [v.tolist() for v in vectors]
