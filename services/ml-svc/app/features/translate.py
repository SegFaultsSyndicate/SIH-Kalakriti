"""services/ml-svc/app/features/translate.py

The Translate RPC's business logic: a thin pass-through to the translation
component. Do-not-translate placeholders are core-svc's `pipeline.go`'s
concern (mask before the call, unmask after) -- this feature does not know
about them and must not touch text content itself.
"""

from __future__ import annotations

import asyncio

from app.models.translation import TranslationModel


class Translator:
    def __init__(self, model: TranslationModel) -> None:
        self._model = model

    async def translate(
        self,
        title: str,
        description: str,
        highlights: list[str],
        source_language: str,
        target_language: str,
    ) -> dict:
        return await asyncio.to_thread(
            self._model.translate, title, description, highlights, source_language, target_language
        )
