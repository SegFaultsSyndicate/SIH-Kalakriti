"""services/ml-svc/app/models/image_quality/real.py

Classical-CV checks, no model weights: corrupt/unreadable, too small, blank,
and blur (Laplacian-variance). `PIL`/`numpy`/`cv2` are imported here, not at
module scope in `__init__.py`, so mock mode never needs them installed.

Deliberately never judges whether the subject matches anything in the craft
ontology -- that is `extract_attributes`'s job downstream, and it already
abstains (empty craft_id) rather than rejecting when nothing matches. A
sharp, well-lit photo of something outside the ontology must pass every
check here.
"""

from __future__ import annotations

import io
import logging

from app.models.image_quality import ImageQualityConfig

log = logging.getLogger(__name__)


class RealImageQuality:
    def __init__(self, cfg: ImageQualityConfig) -> None:
        self._cfg = cfg

    def assess(self, image: bytes) -> dict:
        if not image:
            return _rejection("CORRUPT", "the file is empty")

        import numpy as np
        from PIL import Image

        try:
            decoded = Image.open(io.BytesIO(image))
            decoded.load()
            decoded = decoded.convert("L")
        except Exception:
            log.info("image failed to decode; rejecting as corrupt")
            return _rejection("CORRUPT", "the file could not be read as an image")

        issues = []
        width, height = decoded.size
        if width < self._cfg.min_width_px or height < self._cfg.min_height_px:
            issues.append(
                {
                    "code": "TOO_SMALL",
                    "message": (
                        f"image is {width}x{height}px, below the "
                        f"{self._cfg.min_width_px}x{self._cfg.min_height_px}px minimum"
                    ),
                }
            )

        gray = np.asarray(decoded, dtype=np.uint8)
        stddev = float(gray.std())
        if stddev <= self._cfg.max_blank_stddev:
            issues.append({"code": "BLANK", "message": "image has almost no visual variation"})

        blur_score = _laplacian_variance(gray)
        if blur_score < self._cfg.min_blur_variance:
            issues.append(
                {
                    "code": "BLURRY",
                    "message": f"image is too blurred to be usable (sharpness score {blur_score:.1f})",
                }
            )

        return {"passed": not issues, "issues": issues, "blur_score": blur_score}


def _laplacian_variance(gray) -> float:
    """The canonical no-reference blur score: a sharp image has strong,
    varied second-derivative response everywhere; a blurred one is smoothed
    toward a near-constant Laplacian, so its variance collapses."""
    import cv2

    return float(cv2.Laplacian(gray, cv2.CV_64F).var())


def _rejection(code: str, message: str) -> dict:
    return {"passed": False, "issues": [{"code": code, "message": message}], "blur_score": 0.0}
