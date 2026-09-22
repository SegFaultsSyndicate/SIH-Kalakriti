"""services/ml-svc/app/models/image_lighting/mock.py

No weights, no PIL: an identity passthrough, always "available" (unlike real
mode, mock has no missing-checkpoint case).
"""

from __future__ import annotations


class MockImageLighting:
    def correct(self, image: bytes) -> bytes | None:
        return image
