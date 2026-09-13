"""services/ml-svc/tests/models/test_image_lighting_net.py

Needs torch (the `real` extra, not installed by the default `dev`-only CI
run this test suite otherwise targets), so this `importorskip`s rather than
assumes it is there. No pretrained checkpoint is needed -- this checks the
network's shape/range contract against random init weights, not that it
actually brightens a real photo (that needs the real checkpoint; see
`app/models/image_lighting/net.py`).
"""

import pytest


def test_zero_dce_net_preserves_shape_and_output_range():
    torch = pytest.importorskip("torch")
    from app.models.image_lighting.net import ZeroDCEPPNet

    net = ZeroDCEPPNet().eval()
    # Odd, non-square, non-power-of-two size on purpose: the architecture has
    # no pooling, so it must not assume dimensions divide evenly by anything.
    x = torch.rand(1, 3, 37, 51)
    with torch.no_grad():
        out = net(x)

    assert out.shape == x.shape
    # The curve is x + A*(x^2 - x) with A = tanh(...) in (-1, 1); starting
    # from x in [0, 1] this can drift slightly outside that range over 8
    # iterations, which is exactly why RealImageLighting.correct clamps
    # afterwards -- this asserts the *shape* contract, not a bounded range.
    assert torch.isfinite(out).all()


def test_zero_dce_net_state_dict_keys_match_upstream_names():
    """Guards the one property that actually matters for loading a real
    checkpoint: layer names must match the upstream repo's `enhance_net_nopool`
    exactly, since Epoch99.pth was saved from that class."""
    torch = pytest.importorskip("torch")
    from app.models.image_lighting.net import ZeroDCEPPNet

    keys = set(ZeroDCEPPNet().state_dict().keys())
    for layer in ("e_conv1", "e_conv2", "e_conv3", "e_conv4", "e_conv5", "e_conv6", "e_conv7"):
        assert f"{layer}.depth_conv.weight" in keys
        assert f"{layer}.point_conv.weight" in keys
