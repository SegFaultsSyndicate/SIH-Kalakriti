"""services/ml-svc/app/models/zero_dce.py

Zero-DCE++ (Li et al., "Learning to Enhance Low-Light Image via Zero-Reference
Deep Curve Estimation", the lightweight `enhance_net_nopool` extension of the
original Zero-DCE) ported layer-for-layer from the paper's own reference
implementation (github.com/Li-Chongyi/Zero-DCE_extension), so that the
officially released checkpoint (`Epoch99.pth`) loads into this module's
`state_dict()` unchanged -- the class/attribute names below match upstream
exactly for that reason, not by house style.

DESIGN DECISION: upstream's own `lowlight_test.py` runs inference at a
downsampled resolution (`scale_factor=12`) purely for mobile-class speed, then
upsamples the predicted curve map back to full size. This port always runs at
full resolution instead (no resize step at all): the network has no pooling
layers, so the same weights are valid at any input resolution, and skipping
the resize avoids both an assumption that width/height divide evenly by the
scale factor and any upsample-interpolation softening of the curve map. This
trades some CPU time for robustness/quality; revisit if real profiling shows
it is too slow for the demo's photo sizes (BiRefNet already runs a heavier
network on full-res photos in `enhance_image` today, so this is unlikely to
be the bottleneck, but it has not been measured).

Not imported at module scope from anywhere except
`app/models/image_lighting/real.py`'s lazy net loader, itself only reached
from a real (non-mock) `EnhanceImage` call with `correct_lighting=True` --
mock mode and every other RPC stay free of this import, same discipline as
`app/model_loading.py`.

NOT-YET-VALIDATED: never run against a real checkpoint in this sandbox (no
weights on disk, and this project's standing rule is to never trigger a model
download -- see this service's CLAUDE.md). Zero-DCE++ is not
distributed as a pip package or a Hugging Face repo; the only pretrained
checkpoint is the raw `.pth` file in the paper's GitHub release
(github.com/Li-Chongyi/Zero-DCE_extension, under
`Zero-DCE++/snapshots_Zero_DCE++/Epoch99.pth`). To actually exercise this:
download that file to wherever `ML_SVC_IMAGE_LIGHTING_MODEL` points and set
that env var -- state the download command, don't run it:
  curl -fLo <path> https://raw.githubusercontent.com/Li-Chongyi/Zero-DCE_extension/main/Zero-DCE%2B%2B/snapshots_Zero_DCE%2B%2B/Epoch99.pth
The architecture below is transcribed from the published reference
implementation and believed correct, but has not been confirmed to actually
`load_state_dict()` against that real file, nor to produce a visually-correct
brightened image.
"""

from __future__ import annotations

import torch
import torch.nn.functional as F
from torch import nn


class _DepthwiseSeparableConv(nn.Module):
    """CSDN_Tem in the upstream repo: depthwise 3x3 then pointwise 1x1 -- the
    parameter-count trick that makes Zero-DCE++ ~10K parameters instead of the
    original Zero-DCE's plain-conv ~79K."""

    def __init__(self, in_channels: int, out_channels: int) -> None:
        super().__init__()
        self.depth_conv = nn.Conv2d(
            in_channels, in_channels, kernel_size=3, stride=1, padding=1, groups=in_channels
        )
        self.point_conv = nn.Conv2d(in_channels, out_channels, kernel_size=1, stride=1, padding=0)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.point_conv(self.depth_conv(x))


class ZeroDCEPPNet(nn.Module):
    """DCE-Net: estimates one 3-channel pixel-wise curve-parameter map, applied
    recursively (see `enhance()`) to brighten a low-light image. The zero-
    reference losses from the paper are training-time only; this service only
    ever runs the forward pass against a pretrained checkpoint.
    """

    def __init__(self) -> None:
        super().__init__()
        channels = 32
        self.e_conv1 = _DepthwiseSeparableConv(3, channels)
        self.e_conv2 = _DepthwiseSeparableConv(channels, channels)
        self.e_conv3 = _DepthwiseSeparableConv(channels, channels)
        self.e_conv4 = _DepthwiseSeparableConv(channels, channels)
        self.e_conv5 = _DepthwiseSeparableConv(channels * 2, channels)
        self.e_conv6 = _DepthwiseSeparableConv(channels * 2, channels)
        self.e_conv7 = _DepthwiseSeparableConv(channels * 2, 3)

    def enhance(self, x: torch.Tensor, curve: torch.Tensor) -> torch.Tensor:
        """Applies the learned quadratic curve 8 times, reusing the same
        `curve` map on every iteration -- the parameter-sharing trick that
        lets Zero-DCE++ predict one map instead of the original Zero-DCE's
        eight, at the cost of a fixed (not per-iteration-tuned) curve."""
        for _ in range(8):
            x = x + curve * (torch.pow(x, 2) - x)
        return x

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        x1 = F.relu(self.e_conv1(x))
        x2 = F.relu(self.e_conv2(x1))
        x3 = F.relu(self.e_conv3(x2))
        x4 = F.relu(self.e_conv4(x3))
        x5 = F.relu(self.e_conv5(torch.cat([x3, x4], 1)))
        x6 = F.relu(self.e_conv6(torch.cat([x2, x5], 1)))
        curve = torch.tanh(self.e_conv7(torch.cat([x1, x6], 1)))
        return self.enhance(x, curve)
