"""services/ml-svc/app/models/image_lighting/

The low-light-correction component (Zero-DCE++). Optional, not required:
`correct()` returns `None` when no checkpoint is configured or loading it
failed, and the caller (`features/enhance.py`) skips the operation with a
warning rather than failing the request or startup -- honest degradation,
same pattern as the handloom texture classifier's missing checkpoint.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class ImageLighting(Protocol):
    def correct(self, image: bytes) -> bytes | None: ...


@dataclass(frozen=True)
class ImageLightingConfig:
    """Zero-DCE++ has no pip package or HF repo id -- only a raw .pth from
    the paper's own GitHub release (see `net.py`'s module docstring). Empty
    means "not configured": `correct()` degrades to returning `None` rather
    than failing."""

    checkpoint_path: str = ""

    @classmethod
    def from_env(cls) -> "ImageLightingConfig":
        return cls(checkpoint_path=os.getenv("ML_SVC_IMAGE_LIGHTING_MODEL", ""))


def build(cfg: Config) -> ImageLighting:
    if cfg.mock_mode:
        from app.models.image_lighting.mock import MockImageLighting

        return MockImageLighting()

    from app.models.image_lighting.real import RealImageLighting

    return RealImageLighting(ImageLightingConfig.from_env())
