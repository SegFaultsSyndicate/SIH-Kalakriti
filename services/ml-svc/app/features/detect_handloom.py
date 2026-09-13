"""services/ml-svc/app/features/detect_handloom.py

The DetectHandloom RPC's business logic: fetch the image, hand it to the
handloom-texture component.
"""

from __future__ import annotations

import asyncio

from app.models.handloom_texture import HandloomTexture
from app.models.storage import ObjectStorage


class HandloomDetector:
    def __init__(self, handloom: HandloomTexture, storage: ObjectStorage) -> None:
        self._handloom = handloom
        self._storage = storage

    async def detect(self, object_key: str, declared_thread_count: int) -> dict:
        return await asyncio.to_thread(self._detect_blocking, object_key, declared_thread_count)

    def _detect_blocking(self, object_key: str, declared_thread_count: int) -> dict:
        image = self._storage.get_bytes(object_key)
        return self._handloom.score(image, object_key, declared_thread_count)
