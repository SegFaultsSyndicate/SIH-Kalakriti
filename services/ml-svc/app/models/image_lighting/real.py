"""services/ml-svc/app/models/image_lighting/real.py

Lazily loads the Zero-DCE++ curve-estimation net, once, on first `correct()`
call. `torch`/`PIL` are imported here, not at module scope in `__init__.py`,
so mock mode never needs them installed.
"""

from __future__ import annotations

import io
import logging

from app.models.image_lighting import ImageLightingConfig

log = logging.getLogger(__name__)


class RealImageLighting:
    def __init__(self, cfg: ImageLightingConfig) -> None:
        self._cfg = cfg
        self._net = None
        self._load_failed = False

    def _get_net(self):
        """Lazily loads the net, once. `ML_SVC_IMAGE_LIGHTING_MODEL` unset
        (or a bad checkpoint) must not fail startup or even this call -- it
        just returns None and the caller skips the operation."""
        if self._net is not None or self._load_failed:
            return self._net

        if not self._cfg.checkpoint_path:
            self._load_failed = True
            return None

        try:
            import torch

            from app.models.image_lighting.net import ZeroDCEPPNet

            net = ZeroDCEPPNet()
            state_dict = torch.load(self._cfg.checkpoint_path, map_location="cpu")
            net.load_state_dict(state_dict)
            net.eval()
        except Exception:
            log.exception(
                "failed to load Zero-DCE++ checkpoint at %r; correct_lighting will be a no-op",
                self._cfg.checkpoint_path,
            )
            self._load_failed = True
            return None

        self._net = net
        return net

    def correct(self, image: bytes) -> bytes | None:
        if not image:
            return image

        net = self._get_net()
        if net is None:
            return None

        import numpy as np
        import torch
        from PIL import Image

        decoded = Image.open(io.BytesIO(image)).convert("RGB")
        array = np.asarray(decoded, dtype=np.float32) / 255.0
        tensor = torch.from_numpy(array).permute(2, 0, 1).unsqueeze(0)
        with torch.no_grad():
            enhanced = net(tensor)
        out = enhanced.clamp(0.0, 1.0).squeeze(0).permute(1, 2, 0).numpy()
        result = Image.fromarray((out * 255.0).round().astype(np.uint8))

        buffer = io.BytesIO()
        result.save(buffer, format="PNG")
        return buffer.getvalue()
