"""services/ml-svc/app/models/

The model registry. `build()` returns one object with a method per RPC, loaded
once at startup and shared by every request. Nothing here imports torch or
transformers at module scope: in mock mode those packages need not be installed.
"""

from __future__ import annotations

from typing import Protocol

from app.config import Config


class Models(Protocol):
    """What the servicer needs. Plain Python in, plain Python out: protobuf is
    converted in server.py and never reaches a model."""

    version: str

    async def load(self) -> None: ...

    async def enhance_image(
        self, object_key: str, remove_background: bool, auto_white_balance: bool, upscale: int
    ) -> tuple[str, list[str]]: ...

    async def extract_attributes(
        self, object_keys: list[str], declared_craft_id: str, hint: str
    ) -> dict: ...

    async def generate_description(
        self, attributes: dict, craft_id: str, language: str, artisan_note: str, max_chars: int
    ) -> dict: ...

    async def verify_technique(
        self, object_keys: list[str], claimed: str, craft_id: str
    ) -> dict: ...

    async def detect_handloom(self, object_key: str, declared_thread_count: int) -> dict: ...

    async def embed(self, texts: list[str]) -> list[list[float]]: ...

    async def rerank(self, query: str, candidates: list[tuple[str, str]]) -> list[tuple[str, float]]: ...

    def transcribe(self, object_key: str, language: str, interim: bool): ...


def build(cfg: Config) -> Models:
    """Pick a backend. The import is inside the branch so real-mode packages are
    only needed when real mode is asked for."""
    if cfg.mock_mode:
        from app.models.mock import MockModels

        return MockModels(cfg)

    from app.models.real import RealModels

    return RealModels(cfg)
