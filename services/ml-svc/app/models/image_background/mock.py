"""services/ml-svc/app/models/image_background/mock.py

No weights, no PIL: an identity passthrough. `features/enhance.py` still
records "remove-background" in the operations list whenever it is
requested, regardless of backend -- this component just needs to not raise.
"""

from __future__ import annotations


class MockBackgroundRemover:
    def remove(self, image: bytes) -> bytes:
        return image
