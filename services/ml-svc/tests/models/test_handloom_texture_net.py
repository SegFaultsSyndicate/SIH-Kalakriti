"""services/ml-svc/tests/models/test_handloom_texture_net.py

Needs torch (the `real` extra), so this `importorskip`s rather than assumes
it is there. No pretrained checkpoint exists for this net (see its module
docstring) -- this checks the shape/range contract against random init
weights, not real-world classification accuracy.
"""

import pytest


def test_texture_net_accepts_any_size_and_returns_one_logit_per_image():
    torch = pytest.importorskip("torch")
    from app.models.handloom_texture.net import TextureNet

    net = TextureNet().eval()
    # Two different, non-power-of-two sizes: AdaptiveAvgPool2d must make both
    # work without a shape-mismatch on the final linear layer.
    for batch, height, width in [(1, 512, 512), (2, 97, 143)]:
        x = torch.rand(batch, 1, height, width)
        with torch.no_grad():
            out = net(x)
        assert out.shape == (batch,)
        assert torch.isfinite(out).all()
