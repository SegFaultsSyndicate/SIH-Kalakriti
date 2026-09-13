"""services/ml-svc/tests/models/test_storage_mock.py"""

from app.config import Config
from app.models import storage


def test_mock_get_bytes_is_empty():
    backend = storage.build(Config(mock_mode=True))
    assert backend.get_bytes("any/key.jpg") == b""


def test_mock_put_bytes_does_not_raise():
    backend = storage.build(Config(mock_mode=True))
    backend.put_bytes("any/key.jpg", b"data", "image/jpeg")
