"""services/ml-svc/app/models/image_quality/

The pre-VLM quality gate: corrupt, too-small, blank, or blurred images are
rejected before they ever reach EnhanceImage or the VLM -- this is purely a
signal about the image itself, never about whether its subject matches
anything in the craft ontology (that judgement belongs to `extract_attributes`,
which already abstains rather than rejecting when nothing matches).
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class ImageQuality(Protocol):
    def assess(self, image: bytes) -> dict: ...


@dataclass(frozen=True)
class ImageQualityConfig:
    """Thresholds for the classical-CV checks in `real.py`. Defaults picked
    for a typical phone-camera product photo, not tuned against real listing
    data yet -- `blur_score` is returned in the response precisely so these
    can be re-tuned from observed traffic without a code change."""

    min_width_px: int = 200
    min_height_px: int = 200
    min_blur_variance: float = 100.0
    max_blank_stddev: float = 3.0

    @classmethod
    def from_env(cls) -> "ImageQualityConfig":
        return cls(
            min_width_px=int(os.getenv("ML_SVC_IMAGE_QUALITY_MIN_WIDTH_PX", "200")),
            min_height_px=int(os.getenv("ML_SVC_IMAGE_QUALITY_MIN_HEIGHT_PX", "200")),
            min_blur_variance=float(os.getenv("ML_SVC_IMAGE_QUALITY_MIN_BLUR_VARIANCE", "100.0")),
            max_blank_stddev=float(os.getenv("ML_SVC_IMAGE_QUALITY_MAX_BLANK_STDDEV", "3.0")),
        )


def build(cfg: Config) -> ImageQuality:
    if cfg.mock_mode:
        from app.models.image_quality.mock import MockImageQuality

        return MockImageQuality()

    from app.models.image_quality.real import RealImageQuality

    return RealImageQuality(ImageQualityConfig.from_env())
