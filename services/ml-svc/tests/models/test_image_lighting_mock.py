"""services/ml-svc/tests/models/test_image_lighting_mock.py"""

from app.config import Config
from app.models import image_lighting


def test_mock_is_always_available_and_an_identity_passthrough():
    backend = image_lighting.build(Config(mock_mode=True))
    assert backend.correct(b"pixels") == b"pixels"
