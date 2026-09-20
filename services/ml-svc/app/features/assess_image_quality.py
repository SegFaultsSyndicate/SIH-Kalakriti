"""services/ml-svc/app/features/assess_image_quality.py

The AssessImageQuality RPC's business logic: fetch the image, hand it to the
image-quality component.
"""

from __future__ import annotations

import asyncio

from app.models.image_quality import ImageQuality
from app.models.storage import ObjectStorage


class ImageQualityAssessor:
    def __init__(self, image_quality: ImageQuality, storage: ObjectStorage) -> None:
        self._image_quality = image_quality
        self._storage = storage

    async def assess(self, object_key: str) -> dict:
        return await asyncio.to_thread(self._assess_blocking, object_key)

    def _assess_blocking(self, object_key: str) -> dict:
        image = self._storage.get_bytes(object_key)
        return self._image_quality.assess(image)
