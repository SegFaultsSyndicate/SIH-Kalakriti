"""services/ml-svc/app/features/embed.py

The Embed RPC's business logic: none, really -- it is a thin async wrapper
around one component. Kept as its own feature module anyway, same as every
other RPC, so `server.py` never imports a model component directly.
"""

from __future__ import annotations

import asyncio

from app.models.embedding import TextEmbedder


class Embedder:
    def __init__(self, embedding: TextEmbedder) -> None:
        self._embedding = embedding

    async def embed(self, texts: list[str]) -> list[list[float]]:
        return await asyncio.to_thread(self._embedding.encode, texts)
