"""services/ml-svc/app/models/handloom_texture/

The handloom-vs-powerloom detector. The FFT peak-ratio math is the whole
signal today; an optional texture CNN can sharpen the verdict once trained,
same honest-degradation pattern as image_lighting's checkpoint -- absent
config means it never loads and the FFT ratio decides alone, not a hard
failure. `ML_SVC_HANDLOOM_DETECTION_MODEL` is wired here for when that
checkpoint exists; it does not exist yet (this was dead code before this
refactor too -- see `real.py`).
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class HandloomTexture(Protocol):
    def score(self, image: bytes, object_key: str, declared_thread_count: int) -> dict: ...


@dataclass(frozen=True)
class HandloomTextureConfig:
    checkpoint_path: str = ""

    @classmethod
    def from_env(cls) -> "HandloomTextureConfig":
        return cls(checkpoint_path=os.getenv("ML_SVC_HANDLOOM_DETECTION_MODEL", ""))


def build(cfg: Config) -> HandloomTexture:
    if cfg.mock_mode:
        from app.models.handloom_texture.mock import MockHandloomTexture

        return MockHandloomTexture()

    from app.models.handloom_texture.real import RealHandloomTexture

    return RealHandloomTexture(HandloomTextureConfig.from_env())
