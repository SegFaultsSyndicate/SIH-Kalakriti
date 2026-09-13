"""services/ml-svc/app/features/enhance.py

The EnhanceImage RPC's business logic: a pipeline over two swappable models
(lighting, background) plus two pure PIL ops (white balance, upscale) that
have no "model" to swap -- every deployment does them the same way. The
operations-list bookkeeping is unconditional on the request flags, not on
whether a component actually changed anything: a mock backend still needs to
report what it "would have" applied, same as before this refactor.

The pure ops guard on empty bytes (`if not data: return data`) so they are a
no-op in mock mode, where `storage.get_bytes` never has real content -- this
keeps PIL out of the mock-mode call path without either op needing to know
mock_mode exists.
"""

from __future__ import annotations

import asyncio
import io
import logging

from app.models.image_background import BackgroundRemover
from app.models.image_lighting import ImageLighting
from app.models.storage import ObjectStorage

log = logging.getLogger(__name__)


class ImageEnhancer:
    def __init__(
        self, storage: ObjectStorage, lighting: ImageLighting, background: BackgroundRemover
    ) -> None:
        self._storage = storage
        self._lighting = lighting
        self._background = background

    async def enhance(
        self,
        object_key: str,
        remove_background: bool,
        auto_white_balance: bool,
        upscale: int,
        correct_lighting: bool = False,
    ) -> tuple[str, list[str]]:
        return await asyncio.to_thread(
            self._enhance_blocking, object_key, remove_background, auto_white_balance, upscale, correct_lighting
        )

    def _enhance_blocking(
        self,
        object_key: str,
        remove_background: bool,
        auto_white_balance: bool,
        upscale: int,
        correct_lighting: bool,
    ) -> tuple[str, list[str]]:
        data = self._storage.get_bytes(object_key)
        ops = []

        if auto_white_balance:
            data = _apply_white_balance(data)
            ops.append("auto-white-balance")

        if correct_lighting:
            corrected = self._lighting.correct(data)
            if corrected is not None:
                data = corrected
                ops.append("correct-lighting")
            else:
                log.warning(
                    "correct_lighting requested but the lighting component is unavailable "
                    "(no checkpoint configured); skipping"
                )

        if remove_background:
            data = self._background.remove(data)
            ops.append("remove-background")

        if upscale > 1:
            data = _apply_upscale(data, upscale)
            ops.append(f"upscale-{upscale}x")

        enhanced_key = f"enhanced/{object_key.rsplit('/', 1)[-1].rsplit('.', 1)[0]}.webp"
        self._storage.put_bytes(enhanced_key, _encode_webp(data), "image/webp")
        return enhanced_key, ["denoise", *ops]


def _encode_webp(data: bytes) -> bytes:
    """The final output is always webp, regardless of which ops above ran
    (each of which may hand back PNG-encoded intermediate bytes)."""
    if not data:
        return data
    from PIL import Image

    image = Image.open(io.BytesIO(data)).convert("RGB")
    buffer = io.BytesIO()
    image.save(buffer, format="WEBP", quality=90)
    return buffer.getvalue()


def _apply_white_balance(data: bytes) -> bytes:
    if not data:
        return data
    from PIL import Image, ImageOps

    image = ImageOps.autocontrast(Image.open(io.BytesIO(data)).convert("RGB"), cutoff=1)
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    return buffer.getvalue()


def _apply_upscale(data: bytes, factor: int) -> bytes:
    if not data:
        return data
    from PIL import Image

    image = Image.open(io.BytesIO(data)).convert("RGB")
    image = image.resize((image.width * factor, image.height * factor), Image.LANCZOS)
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    return buffer.getvalue()
