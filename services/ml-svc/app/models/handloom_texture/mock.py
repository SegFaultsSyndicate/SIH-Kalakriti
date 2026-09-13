"""services/ml-svc/app/models/handloom_texture/mock.py

Deterministic, seeded off the object key -- no numpy, no FFT.
"""

from __future__ import annotations

import hashlib
import random


def _rng(*parts: object) -> random.Random:
    digest = hashlib.sha256("|".join(str(p) for p in parts).encode()).digest()
    return random.Random(int.from_bytes(digest[:8], "big"))


class MockHandloomTexture:
    def score(self, image: bytes, object_key: str, declared_thread_count: int) -> dict:
        rng = _rng("loom", object_key, declared_thread_count)
        # Powerloom weave is far more periodic, so a high peak-to-median ratio is
        # the machine-made case; the mock keeps that relationship intact.
        ratio = 1.6 + rng.random() * 6.0
        is_handloom = ratio < 4.0
        return {
            "is_handloom": is_handloom,
            "confidence": 0.6 + rng.random() * 0.35,
            "fft_peak_ratio": ratio,
            "explanation": (
                f"Thread spacing varies across the crop (peak-to-median {ratio:.1f}), which is "
                "what a hand-thrown shuttle looks like."
                if is_handloom
                else f"Thread spacing is highly regular (peak-to-median {ratio:.1f}), which points to a powerloom."
            ),
        }
