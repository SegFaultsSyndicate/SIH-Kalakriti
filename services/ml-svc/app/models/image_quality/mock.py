"""services/ml-svc/app/models/image_quality/mock.py

No PIL, no cv2: always passes. Mock mode has no real content anywhere
(`app.models.storage`'s mock backend always returns b""), so there is
nothing to reject here by design -- same "mock has nothing to guard against"
pattern as `image_lighting`'s identity passthrough.
"""

from __future__ import annotations


class MockImageQuality:
    def assess(self, image: bytes) -> dict:
        return {"passed": True, "issues": [], "blur_score": 500.0}
