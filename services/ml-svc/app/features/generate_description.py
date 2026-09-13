"""services/ml-svc/app/features/generate_description.py

The GenerateDescription RPC's business logic: template-first,
model-polish-second. A deterministic sentence built only from the given
attributes is the structural grounding guarantee -- `attribute_keys_used` is
exactly the set of keys the template drew from, not the model's say-so about
what it used. The vlm component only gets to polish tone from there, and its
rewrite is kept only if `_is_grounded` confirms every cited value survived
-- true for both a real rewrite and mock's own passthrough, which is
grounded by construction.
"""

from __future__ import annotations

import asyncio
import logging

from app.features.templates import (
    _is_grounded,
    _template_highlights,
    _template_keywords,
    _template_sentence,
    _template_title,
)
from app.models.vlm import VisionLanguageModel

log = logging.getLogger(__name__)


class DescriptionGenerator:
    def __init__(self, vlm: VisionLanguageModel) -> None:
        self._vlm = vlm

    async def generate(
        self, attributes: dict, craft_id: str, language: str, artisan_note: str, max_chars: int
    ) -> dict:
        return await asyncio.to_thread(
            self._generate_blocking, attributes, craft_id, language, artisan_note, max_chars
        )

    def _generate_blocking(
        self, attributes: dict, craft_id: str, language: str, artisan_note: str, max_chars: int
    ) -> dict:
        facts = {k: v[0] for k, v in attributes.items() if isinstance(v, tuple) and v[0]}
        used = [k for k in ("material", "technique", "colours", "motifs") if facts.get(k)]
        template = _template_sentence(facts, craft_id)
        description = self._polish(template, facts, language, max_chars)
        if artisan_note:
            description = f"{description} In the maker's words: {artisan_note.strip()}"
        if max_chars > 0:
            description = description[:max_chars]

        return {
            "title": _template_title(facts, craft_id)[:120],
            "description": description,
            "highlights": _template_highlights(facts),
            "keywords": _template_keywords(facts, craft_id),
            "attribute_keys_used": used,
            "language": language,
        }

    def _polish(self, template: str, facts: dict, language: str, max_chars: int) -> str:
        try:
            rewrite = self._vlm.polish(template, facts, language, max_chars).strip()
        except Exception:
            log.exception("polish pass failed, falling back to the template sentence")
            return template

        if not _is_grounded(rewrite, facts):
            log.warning("polish rewrite dropped or altered a cited attribute value; using the template")
            return template
        return rewrite
