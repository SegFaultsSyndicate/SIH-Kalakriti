"""services/ml-svc/tests/models/test_image_quality_mock.py"""

from app.config import Config
from app.models import image_quality


def test_mock_always_passes():
    backend = image_quality.build(Config(mock_mode=True))
    verdict = backend.assess(b"")
    assert verdict["passed"]
    assert verdict["issues"] == []
