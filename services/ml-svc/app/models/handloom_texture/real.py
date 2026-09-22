"""services/ml-svc/app/models/handloom_texture/real.py

FFT peak-ratio detector plus an optional texture CNN. `numpy` is imported
here, not at module scope in `__init__.py`, so mock mode never needs it
installed.

`ML_SVC_HANDLOOM_DETECTION_MODEL` unset (the default) means `_get_net()`
never constructs `TextureNet` at all -- same honest-degradation pattern as
`image_lighting`'s Zero-DCE++ checkpoint: a missing/bad checkpoint must not
fail startup or even this call, it just means the FFT ratio decides alone.
Unlike `image_lighting`, there is no pretrained checkpoint anywhere to point
this at yet (see `net.py`'s module docstring) -- wiring the loader now means
`RealHandloomTexture` is ready the moment a from-scratch-trained checkpoint
exists, without another round of surgery here.

DATASET BOOTSTRAP, for whoever trains that first checkpoint: don't start
from a blank labelling effort. `score()`'s own `fft_peak_ratio` is already a
decent weak label -- run it over a pile of unlabelled weave close-ups (real
listing photos once artisans are uploading them, or scraped/purchased
sample shots in the meantime) and treat the confident tails (ratio well
below or well above the 4.0 threshold) as provisional labels; a human only
needs to adjudicate the ambiguous middle, which is also exactly the band
this CNN is meant to help with. That is a much smaller labelling job than
starting from zero, and it means the CNN's job is "resolve what FFT alone
can't," not "replace FFT," which is exactly how `score()` already combines
the two (`is_handloom = peak_ratio < 4.0 and texture_score < 0.5`).
"""

from __future__ import annotations

import io
import logging

from app.models.handloom_texture import HandloomTextureConfig

log = logging.getLogger(__name__)


class RealHandloomTexture:
    def __init__(self, cfg: HandloomTextureConfig) -> None:
        self._cfg = cfg
        self._texture_net = None
        self._load_failed = False

    def _get_net(self):
        """Lazily loads the net, once. See `image_lighting/real.py`'s
        `_get_net` -- identical contract: unset or bad checkpoint must not
        fail startup or even this call, just returns None."""
        if self._texture_net is not None or self._load_failed:
            return self._texture_net

        if not self._cfg.checkpoint_path:
            self._load_failed = True
            return None

        try:
            import torch

            from app.models.handloom_texture.net import TextureNet

            net = TextureNet()
            state_dict = torch.load(self._cfg.checkpoint_path, map_location="cpu")
            net.load_state_dict(state_dict)
            net.eval()
        except Exception:
            log.exception(
                "failed to load handloom texture checkpoint at %r; "
                "the FFT peak-ratio will decide alone",
                self._cfg.checkpoint_path,
            )
            self._load_failed = True
            return None

        self._texture_net = net
        return net

    def score(self, image: bytes, object_key: str, declared_thread_count: int) -> dict:
        import numpy as np
        from PIL import Image

        decoded = Image.open(io.BytesIO(image)).convert("RGB")
        # A centre crop, greyscale: the weave is what matters, not the drape.
        side = min(decoded.size)
        left, top = (decoded.width - side) // 2, (decoded.height - side) // 2
        crop = decoded.crop((left, top, left + side, top + side)).resize((512, 512)).convert("L")
        raw_pixels = np.asarray(crop, dtype=np.float64)
        pixels = raw_pixels - raw_pixels.mean()

        # A powerloom lays every pick at the same spacing, so its spectrum has one
        # sharp peak; a hand-thrown shuttle smears the energy across neighbouring
        # frequencies. The ratio of the strongest peak to the median is the whole
        # signal.
        spectrum = np.abs(np.fft.fftshift(np.fft.fft2(pixels * np.hanning(512)[:, None] * np.hanning(512))))
        centre = 512 // 2
        spectrum[centre - 3 : centre + 4, centre - 3 : centre + 4] = 0  # drop the DC blob
        peak_ratio = float(spectrum.max() / (np.median(spectrum[spectrum > 0]) or 1.0))

        # raw_pixels, not the FFT's mean-subtracted pixels -- the net expects
        # ordinary [0, 1]-normalised greyscale, not a zero-centred FFT input.
        texture_score = self._texture_cnn_score(raw_pixels)
        # ponytail: fixed threshold from the pilot set. Fit it properly once there
        # are labelled clusters; the ratio is returned so the Go side can re-judge.
        is_handloom = peak_ratio < 4.0 and texture_score < 0.5
        confidence = min(0.99, abs(peak_ratio - 4.0) / 4.0 * 0.5 + 0.5)

        return {
            "is_handloom": is_handloom,
            "confidence": float(confidence),
            "fft_peak_ratio": peak_ratio,
            "explanation": (
                f"Thread spacing varies across the crop (peak-to-median {peak_ratio:.1f}), which is "
                "what a hand-thrown shuttle looks like."
                if is_handloom
                else f"Thread spacing is highly regular (peak-to-median {peak_ratio:.1f}), which points to a powerloom."
            ),
        }

    def _texture_cnn_score(self, raw_pixels) -> float:
        """Probability the crop looks machine-made, from `raw_pixels`: the
        greyscale crop as ordinary [0, 255] values, NOT the FFT's
        mean-subtracted array (see `score()`'s comment on that).

        The classifier is optional: without its checkpoint (`_get_net()`
        returns None) the FFT ratio decides alone, the honest degradation
        rather than a hard failure.
        """
        net = self._get_net()
        if net is None:
            return 0.0
        import torch

        with torch.no_grad():
            patches = torch.tensor(raw_pixels, dtype=torch.float32)[None, None] / 255.0
            return float(torch.sigmoid(net(patches)).mean())
