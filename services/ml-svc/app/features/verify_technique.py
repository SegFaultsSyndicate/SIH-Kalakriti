"""services/ml-svc/app/features/verify_technique.py

The VerifyTechnique RPC's business logic: fetch each media item, ask the vlm
component what it observed, then cross-check the model's own "matches"
verdict against a pure word-overlap comparison of `observed` vs `claimed` --
a model can say matches=true while still describing something different
than what it named; `_same_technique` catches that regardless of backend.
"""

from __future__ import annotations

import asyncio

from app.features.templates import _confidence, _same_technique
from app.models.storage import ObjectStorage
from app.models.vlm import VisionLanguageModel


class TechniqueVerifier:
    def __init__(self, vlm: VisionLanguageModel, storage: ObjectStorage, video_frames: int = 8) -> None:
        self._vlm = vlm
        self._storage = storage
        self._video_frames = video_frames

    async def verify(self, object_keys: list[str], claimed: str, craft_id: str) -> dict:
        return await asyncio.to_thread(self._verify_blocking, object_keys, claimed, craft_id)

    def _verify_blocking(self, object_keys: list[str], claimed: str, craft_id: str) -> dict:
        media = [self._storage.get_bytes(k) for k in object_keys]
        raw = self._vlm.observe_technique(media, object_keys, claimed, craft_id, self._video_frames)
        observed = raw.get("observed", "")
        return {
            "claimed": claimed,
            "observed": observed,
            "matches": bool(raw.get("matches")) and _same_technique(observed, claimed),
            "confidence": _confidence(raw, "confidence"),
            "explanation": raw.get("explanation", ""),
        }
