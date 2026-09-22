"""services/ml-svc/tests/models/test_reranking_mock.py"""

from app.config import Config
from app.models import reranking


def test_scores_every_text_and_favours_lexical_overlap():
    backend = reranking.build(Config(mock_mode=True))
    scores = backend.score("ajrakh stole", ["ajrakh cotton stole", "brass lamp"])

    assert len(scores) == 2
    assert scores[0] > scores[1]


def test_scores_empty_candidates():
    backend = reranking.build(Config(mock_mode=True))
    assert backend.score("q", []) == []
