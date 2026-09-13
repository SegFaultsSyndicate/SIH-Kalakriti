"""services/ml-svc/app/models/image_background/

The background-removal component. Operates on already-fetched image bytes,
not object keys -- fetching is `features/enhance.py`'s job via the storage
component, kept separate so this component stays swappable on its own.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class BackgroundRemover(Protocol):
    """Empty bytes in means "nothing fetched" (mock mode's storage never has
    real content) and must come back out unchanged, never raise."""

    def remove(self, image: bytes) -> bytes: ...


@dataclass(frozen=True)
class ImageBackgroundConfig:
    model: str = "birefnet-general"

    @classmethod
    def from_env(cls) -> "ImageBackgroundConfig":
        return cls(model=os.getenv("ML_SVC_IMAGE_BACKGROUND_MODEL", "birefnet-general"))


def build(cfg: Config) -> BackgroundRemover:
    if cfg.mock_mode:
        from app.models.image_background.mock import MockBackgroundRemover

        return MockBackgroundRemover()

    from app.models.image_background.real import RealBackgroundRemover

    return RealBackgroundRemover(ImageBackgroundConfig.from_env())
