"""services/ml-svc/tests/models/test_handloom_texture_real.py

The one real-mode behaviour that needs no torch: an unconfigured checkpoint
degrades honestly to "FFT ratio decides alone" instead of raising. Mirrors
tests/models/test_image_lighting_real.py.
"""

from app.models.handloom_texture import HandloomTextureConfig
from app.models.handloom_texture.real import RealHandloomTexture


def test_no_checkpoint_configured_scores_without_importing_torch():
    handloom = RealHandloomTexture(HandloomTextureConfig(checkpoint_path=""))
    assert handloom._get_net() is None
    # Cached: a second call must not re-attempt loading.
    assert handloom._get_net() is None
