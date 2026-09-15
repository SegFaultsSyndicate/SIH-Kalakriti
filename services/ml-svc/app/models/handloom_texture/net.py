"""services/ml-svc/app/models/handloom_texture/net.py

A small CNN that scores a weave close-up as machine-regular (powerloom) vs
hand-thrown (handloom), meant to sharpen `real.py`'s FFT peak-ratio verdict
in the ambiguous middle band, not replace it.

Deliberately NOT a port of any published architecture, unlike
`image_lighting/net.py`'s Zero-DCE++: there is no public handloom-vs-powerloom
checkpoint to match layer-for-layer. Checked before writing this -- two
papers doing exactly this binary classification (Barbhuiya & Karmakar,
"Handloomed fabrics recognition with deep learning", Sci Rep 2024,
gamucha towels; and a 2024 NMITCON paper on Mekhela Sador saris) both built
their own private, unreleased datasets (100-200 physical fabric samples,
phone-camera macro shots, heavily augmented to 17k+ images) and both trained
their own small/modified nets from scratch rather than reusing a published
checkpoint -- no such checkpoint exists to reuse. The first paper is the
more useful data point: it explicitly benchmarks ImageNet-pretrained
VGG16/19, ResNet50, InceptionV3, and DenseNet201 against a small custom net
on this exact task, and the pretrained backbones did WORSE (50-92% val
accuracy, several with poor generalisation) than the ~11M-param custom net
(94-98%). That is not surprising in hindsight: weave regularity is a local,
frequency-domain property -- literally what `real.py`'s FFT peak-ratio
already measures analytically -- not the object-shape semantics ImageNet
pretraining teaches. So this net is sized for training from scratch on a
small, task-specific dataset, not for loading a pretrained backbone.

Reuses `image_lighting/net.py`'s depthwise-separable-conv trick for the same
reason it was chosen there: a few thousand parameters, not millions, both
because a first from-scratch dataset here will likely stay small for a
while (see `real.py`'s module docstring for how to bootstrap one) and
because a heavier net has more spare capacity to memorise a small dataset's
incidental texture instead of the periodicity that actually distinguishes
handloom from powerloom.

Global average pooling instead of a fixed-size flatten: like Zero-DCE++,
this must accept whatever crop size `real.py` hands it (currently 512x512,
see `HandloomTexture.score`) without a shape-mismatch on the final linear
layer if that crop size ever changes.

NOT-YET-VALIDATED: no checkpoint exists to load, so `RealHandloomTexture`
never trains or fine-tunes this itself -- `ML_SVC_HANDLOOM_DETECTION_MODEL`
unset (the default) means `_get_net()` never even constructs it, and the FFT
ratio decides alone, same honest degradation as `image_lighting`. Training
this from scratch on a real dataset, then pointing that env var at the
resulting checkpoint, is future work.
"""

from __future__ import annotations

import torch
import torch.nn.functional as F
from torch import nn


class _DownBlock(nn.Module):
    """Depthwise 3x3 stride-2 (downsample) then pointwise 1x1 (mix channels)
    -- see `image_lighting/net.py`'s `_DepthwiseSeparableConv` docstring for
    why this shape is parameter-cheap; the only difference here is the
    stride, since this net needs to shrink spatial size, unlike Zero-DCE++'s
    curve-estimation net which must preserve it."""

    def __init__(self, in_channels: int, out_channels: int) -> None:
        super().__init__()
        self.depth_conv = nn.Conv2d(
            in_channels, in_channels, kernel_size=3, stride=2, padding=1, groups=in_channels
        )
        self.point_conv = nn.Conv2d(in_channels, out_channels, kernel_size=1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return F.relu(self.point_conv(self.depth_conv(x)))


class TextureNet(nn.Module):
    """Binary weave-regularity classifier over a single-channel greyscale
    crop of any size. `forward` returns one raw logit per image (positive =
    looks machine-regular); `real.py` applies `sigmoid` itself, matching how
    the rest of this component reports probabilities."""

    def __init__(self) -> None:
        super().__init__()
        self.stem = nn.Conv2d(1, 16, kernel_size=3, padding=1)
        self.down1 = _DownBlock(16, 32)
        self.down2 = _DownBlock(32, 64)
        self.down3 = _DownBlock(64, 64)
        self.pool = nn.AdaptiveAvgPool2d(1)
        self.fc = nn.Linear(64, 1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        x = F.relu(self.stem(x))
        x = self.down1(x)
        x = self.down2(x)
        x = self.down3(x)
        x = self.pool(x).flatten(1)
        return self.fc(x).squeeze(-1)
