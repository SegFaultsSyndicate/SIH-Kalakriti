"""services/ml-svc/tests/models/test_handloom_texture_mock.py"""

from app.config import Config
from app.models import handloom_texture


def test_verdict_agrees_with_its_own_ratio():
    backend = handloom_texture.build(Config(mock_mode=True))
    for key in ("a.jpg", "b.jpg", "c.jpg", "d.jpg"):
        verdict = backend.score(b"", key, 120)
        assert verdict["is_handloom"] == (verdict["fft_peak_ratio"] < 4.0)
        assert f"{verdict['fft_peak_ratio']:.1f}" in verdict["explanation"]
