"""services/ml-svc/app/registry.py

Loads every swappable model component once, at startup, and hands back a
plain `Registry` of instances that `server.py` builds `features/*.py`
objects from. This is the one file that names every component -- deliberately
explicit rather than a directory-scanning plugin system, so "what backends
does this service load" is answered by reading one short function.

Adding a component: add its module to the imports below, add a field to
`Registry`, add a `load(...)` call in `load_all`. Nothing else in this file
needs to change, and nothing outside `app/models/<component>/` needs to
change either.
"""

from __future__ import annotations

import asyncio
import inspect
import os
from dataclasses import dataclass
from typing import Any

from app.config import Config
from app.models import (
    embedding,
    handloom_texture,
    image_background,
    image_lighting,
    image_quality,
    reranking,
    storage,
    translation,
    vlm,
    voice_transcription,
)
from app.models.embedding import TextEmbedder
from app.models.handloom_texture import HandloomTexture
from app.models.image_background import BackgroundRemover
from app.models.image_lighting import ImageLighting
from app.models.image_quality import ImageQuality
from app.models.reranking import TextReranker
from app.models.storage import ObjectStorage
from app.models.translation import TranslationModel
from app.models.vlm import VisionLanguageModel
from app.models.voice_transcription import VoiceTranscriber


@dataclass(frozen=True)
class Registry:
    storage: ObjectStorage
    background: BackgroundRemover
    lighting: ImageLighting
    image_quality: ImageQuality
    vlm: VisionLanguageModel
    embedder: TextEmbedder
    reranker: TextReranker
    handloom_texture: HandloomTexture
    transcriber: VoiceTranscriber
    translator: TranslationModel


async def load(component_module: Any, cfg: Config) -> Any:
    """Runs one component's `build(cfg)` off the event loop -- real mode's
    eager loads (embedding, reranking) are genuinely slow, and the gRPC
    server is already listening (reporting NOT_SERVING) while this runs, so
    a synchronous call here would block every other in-flight request,
    including health checks. A component whose `build` is itself a coroutine
    function is awaited directly instead, for a future component that needs
    real async I/O (e.g. an API warm-up ping) rather than CPU-bound loading.
    """
    if inspect.iscoroutinefunction(component_module.build):
        return await component_module.build(cfg)
    return await asyncio.to_thread(component_module.build, cfg)


def _apply_model_cache_dir(cfg: Config) -> None:
    """Translates the one generic `Config.model_cache_dir` into whatever env
    var each vendor library underneath a component actually reads -- the only
    place in this service that needs to know those vendor-specific names
    (`HF_HOME` for the four huggingface_hub-backed components: embedding,
    reranking, vlm, translation; `U2NET_HOME` for rembg, behind
    image_background). Keeping this translation here, not in docker-compose.yml
    or any one component, is what lets the deployment-facing env surface and
    every component's own config stay vendor-agnostic. `setdefault` so an
    operator who already set one of these directly is never overridden.
    """
    if not cfg.model_cache_dir:
        return
    os.environ.setdefault("HF_HOME", cfg.model_cache_dir)
    os.environ.setdefault("U2NET_HOME", os.path.join(cfg.model_cache_dir, "rembg"))


async def load_all(cfg: Config) -> Registry:
    _apply_model_cache_dir(cfg)
    (
        storage_backend,
        background,
        lighting,
        quality,
        vlm_backend,
        embedder,
        reranker,
        handloom,
        transcriber,
        translator,
    ) = await asyncio.gather(
        load(storage, cfg),
        load(image_background, cfg),
        load(image_lighting, cfg),
        load(image_quality, cfg),
        load(vlm, cfg),
        load(embedding, cfg),
        load(reranking, cfg),
        load(handloom_texture, cfg),
        load(voice_transcription, cfg),
        load(translation, cfg),
    )
    return Registry(
        storage=storage_backend,
        background=background,
        lighting=lighting,
        image_quality=quality,
        vlm=vlm_backend,
        embedder=embedder,
        reranker=reranker,
        handloom_texture=handloom,
        transcriber=transcriber,
        translator=translator,
    )
