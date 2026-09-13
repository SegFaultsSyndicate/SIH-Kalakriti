"""services/ml-svc/tests/features/test_enhance.py

Fakes stand in for storage/lighting/background so this exercises the
feature's own bookkeeping (the operations list, the output key, the
always-webp final content type) with no PIL/torch needed at all.
"""

from app.features.enhance import ImageEnhancer


class FakeStorage:
    def __init__(self) -> None:
        self.written: dict[str, tuple[bytes, str]] = {}

    def get_bytes(self, object_key: str) -> bytes:
        return b""

    def put_bytes(self, object_key: str, data: bytes, content_type: str) -> None:
        self.written[object_key] = (data, content_type)


class FakeLighting:
    def __init__(self, available: bool = True) -> None:
        self.available = available

    def correct(self, image: bytes) -> bytes | None:
        return image if self.available else None


class FakeBackground:
    def remove(self, image: bytes) -> bytes:
        return image


async def test_enhance_names_the_operations_it_applied():
    storage = FakeStorage()
    enhancer = ImageEnhancer(storage, FakeLighting(), FakeBackground())

    key, ops = await enhancer.enhance("artisans/x/y.jpg", True, True, 2, True)

    assert key == "enhanced/y.webp"
    assert ops == ["denoise", "auto-white-balance", "correct-lighting", "remove-background", "upscale-2x"]
    assert storage.written[key][1] == "image/webp"


async def test_enhance_correct_lighting_defaults_off():
    enhancer = ImageEnhancer(FakeStorage(), FakeLighting(), FakeBackground())
    _, ops = await enhancer.enhance("artisans/x/y.jpg", False, False, 1, False)
    assert "correct-lighting" not in ops


async def test_enhance_skips_lighting_when_component_unavailable():
    enhancer = ImageEnhancer(FakeStorage(), FakeLighting(available=False), FakeBackground())
    _, ops = await enhancer.enhance("artisans/x/y.jpg", False, False, 1, True)
    assert "correct-lighting" not in ops
