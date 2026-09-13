"""services/ml-svc/tests/features/test_embed.py

Uses a hand-written fake, not the real component's mock -- exercising this
feature does not need app.models.embedding at all.
"""

from app.features.embed import Embedder


class FakeEmbedder:
    def encode(self, texts: list[str]) -> list[list[float]]:
        return [[float(len(t))] for t in texts]


async def test_embed_delegates_to_the_component():
    embedder = Embedder(FakeEmbedder())
    assert await embedder.embed(["ab", "abc"]) == [[2.0], [3.0]]
