"""services/ml-svc/app/models/embedding/

The text-embedding component. Swappable independently of reranking, the VLM,
or anything else: `ML_SVC_TEXT_EMBEDDING_MODEL` names a SentenceTransformer
repo id today; a future backend (a hosted embeddings API, say) implements the
same `TextEmbedder` Protocol with no change to `features/embed.py`.

`backend` picks the sentence-transformers execution backend, not a different
component -- "onnx" (default) runs this same repo id through ONNX Runtime
instead of eager PyTorch. sentence-transformers exports the checkpoint to
ONNX on first load if the repo has no pre-exported `onnx/model.onnx` (needs
the `onnx` extra -- see pyproject.toml) and caches the export next to the
torch weights in the same HF cache entry, so this is a one-time cost per
repo id, not a separate download. Set `ML_SVC_TEXT_EMBEDDING_BACKEND=torch`
to fall back to eager mode.
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
    # Overrides app.model_loading.resolve_device()'s auto-detected device.
    # Unset (the default) keeps auto-detection; set to "cpu" to free a small
    # GPU's VRAM for a component that needs it more (see vlm's 4-bit
    # quantization) -- this model is small enough to run fine on CPU.
    device: str | None = None
    # "onnx" or "torch" -- see this module's docstring.
    backend: str = "onnx"

    @classmethod
    def from_env(cls) -> "EmbeddingConfig":
        return cls(
            model=os.getenv("ML_SVC_TEXT_EMBEDDING_MODEL", "intfloat/multilingual-e5-base"),
            device=os.getenv("ML_SVC_TEXT_EMBEDDING_DEVICE") or None,
            backend=os.getenv("ML_SVC_TEXT_EMBEDDING_BACKEND", "onnx"),
        )


def build(cfg: Config) -> TextEmbedder:
    if cfg.mock_mode:
        from app.models.embedding.mock import MockTextEmbedder

        return MockTextEmbedder()

    from app.models.embedding.real import RealTextEmbedder

    return RealTextEmbedder(EmbeddingConfig.from_env())
