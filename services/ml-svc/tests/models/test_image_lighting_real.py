"""services/ml-svc/tests/models/test_image_lighting_real.py

The one real-mode behaviour that needs no torch: an unconfigured checkpoint
degrades honestly to "unavailable" instead of raising.
"""

from app.models.image_lighting import ImageLightingConfig
from app.models.image_lighting.real import RealImageLighting


def test_no_checkpoint_configured_returns_none_without_importing_torch():
    lighting = RealImageLighting(ImageLightingConfig(checkpoint_path=""))
    assert lighting.correct(b"pixels") is None
    # Cached: a second call must not re-attempt loading.
    assert lighting.correct(b"pixels") is None
