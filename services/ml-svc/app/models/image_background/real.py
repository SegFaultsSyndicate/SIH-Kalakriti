"""services/ml-svc/app/models/image_background/real.py

rembg/BiRefNet-backed background removal. `rembg`/`PIL` are imported here,
not at module scope in `__init__.py`, so mock mode never needs them installed.
"""

from __future__ import annotations

import io

from app.models.image_background import ImageBackgroundConfig


class RealBackgroundRemover:
    def __init__(self, cfg: ImageBackgroundConfig) -> None:
        self._cfg = cfg
        self._session = None

    def remove(self, image: bytes) -> bytes:
        if not image:
            return image

        from PIL import Image
        from rembg import new_session, remove

        if self._session is None:
            # BiRefNet keeps the fringe on a dupatta instead of eating it.
            self._session = new_session(self._cfg.model)

        decoded = Image.open(io.BytesIO(image)).convert("RGB")
        cut = remove(decoded, session=self._session)
        flat = Image.new("RGB", cut.size, (245, 245, 245))
        flat.paste(cut, mask=cut.split()[-1])

        buffer = io.BytesIO()
        flat.save(buffer, format="PNG")
        return buffer.getvalue()
