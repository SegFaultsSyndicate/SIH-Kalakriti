"""services/ml-svc/tests/models/test_image_background_mock.py"""

from app.config import Config
from app.models import image_background


def test_mock_is_an_identity_passthrough():
    backend = image_background.build(Config(mock_mode=True))
    assert backend.remove(b"") == b""
    assert backend.remove(b"pixels") == b"pixels"
