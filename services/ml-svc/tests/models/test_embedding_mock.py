"""services/ml-svc/tests/models/test_embedding_mock.py"""

import math

from app.config import Config
from app.models import embedding


def _backend():
    return embedding.build(Config(mock_mode=True))


def test_embeddings_are_768_dim_unit_vectors_and_stable():
    backend = _backend()
    vectors = backend.encode(["ajrakh dupatta", "अजरख दुपट्टा"])

    for v in vectors:
        assert len(v) == 768
        assert math.isclose(math.sqrt(sum(x * x for x in v)), 1.0, rel_tol=1e-9)

    assert backend.encode(["ajrakh dupatta"])[0] == vectors[0]
    assert vectors[0] != vectors[1]
