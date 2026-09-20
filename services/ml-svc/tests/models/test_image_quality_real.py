"""services/ml-svc/tests/models/test_image_quality_real.py

Needs PIL/numpy/cv2 (the `real` extra, not installed by the default `dev`-only
CI run this test suite otherwise targets), so this `importorskip`s rather than
assumes they are there -- same convention as `test_image_lighting_net.py`.
"""

import io

import pytest

from app.models.image_quality import ImageQualityConfig


def _jpeg(image) -> bytes:
    buffer = io.BytesIO()
    image.save(buffer, format="JPEG", quality=95)
    return buffer.getvalue()


def test_corrupt_bytes_are_rejected():
    pytest.importorskip("PIL")
    from app.models.image_quality.real import RealImageQuality

    verdict = RealImageQuality(ImageQualityConfig()).assess(b"not an image")
    assert not verdict["passed"]
    assert verdict["issues"] == [{"code": "CORRUPT", "message": "the file could not be read as an image"}]


def test_empty_bytes_are_rejected_without_decoding():
    from app.models.image_quality.real import RealImageQuality

    verdict = RealImageQuality(ImageQualityConfig()).assess(b"")
    assert not verdict["passed"]
    assert verdict["issues"][0]["code"] == "CORRUPT"


def test_too_small_image_is_rejected():
    pytest.importorskip("PIL")
    pytest.importorskip("cv2")
    from PIL import Image

    from app.models.image_quality.real import RealImageQuality

    backend = RealImageQuality(ImageQualityConfig(min_width_px=200, min_height_px=200, max_blank_stddev=0))
    tiny = Image.new("RGB", (50, 50), color=(120, 60, 30))
    verdict = backend.assess(_jpeg(tiny))
    assert not verdict["passed"]
    assert any(i["code"] == "TOO_SMALL" for i in verdict["issues"])


def test_blank_image_is_rejected():
    pytest.importorskip("PIL")
    pytest.importorskip("cv2")
    from PIL import Image

    from app.models.image_quality.real import RealImageQuality

    backend = RealImageQuality(ImageQualityConfig(min_width_px=10, min_height_px=10))
    blank = Image.new("RGB", (400, 400), color=(128, 128, 128))
    verdict = backend.assess(_jpeg(blank))
    assert not verdict["passed"]
    assert any(i["code"] == "BLANK" for i in verdict["issues"])


def test_heavily_blurred_image_is_rejected():
    pytest.importorskip("cv2")
    np = pytest.importorskip("numpy")
    from PIL import Image, ImageFilter

    from app.models.image_quality.real import RealImageQuality

    rng = np.random.default_rng(0)
    noise = Image.fromarray(rng.integers(0, 255, (400, 400, 3), dtype=np.uint8))
    blurred = noise.filter(ImageFilter.GaussianBlur(radius=12))

    backend = RealImageQuality(ImageQualityConfig(min_width_px=10, min_height_px=10, max_blank_stddev=0))
    verdict = backend.assess(_jpeg(blurred))
    assert not verdict["passed"]
    assert any(i["code"] == "BLURRY" for i in verdict["issues"])


def test_sharp_high_contrast_image_passes():
    pytest.importorskip("cv2")
    np = pytest.importorskip("numpy")
    from PIL import Image

    from app.models.image_quality.real import RealImageQuality

    # A checkerboard has sharp edges everywhere -- high Laplacian variance,
    # not blank, not too small.
    board = np.indices((400, 400)).sum(axis=0) % 40 < 20
    checkerboard = Image.fromarray((board * 255).astype(np.uint8)).convert("RGB")

    backend = RealImageQuality(ImageQualityConfig(min_width_px=10, min_height_px=10))
    verdict = backend.assess(_jpeg(checkerboard))
    assert verdict["passed"]
    assert verdict["issues"] == []
    assert verdict["blur_score"] > 0
