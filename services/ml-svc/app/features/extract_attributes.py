"""services/ml-svc/app/features/extract_attributes.py

The ExtractAttributes RPC's business logic: fetch each image, ask the vlm
component for a raw proposal, then enforce the two structural grounding
guarantees that apply regardless of which model produced the proposal --
the craft id must be one this ontology actually has, and once the craft is
known, material/technique are constrained to *that craft's own* seeded
vocabulary. A model asked nicely to stay on-list still occasionally invents
("silk" for a terracotta piece); this is the structural guard, not a
prompting hope.
"""

from __future__ import annotations

import asyncio

from app.config import load_craft_allowlist, load_craft_vocab
from app.features.templates import _closed_vocab, _confidence, _strings, _vocab_hint
from app.models.storage import ObjectStorage
from app.models.vlm import VisionLanguageModel


class AttributeExtractor:
    def __init__(self, vlm: VisionLanguageModel, storage: ObjectStorage, craft_allowlist_path) -> None:
        self._vlm = vlm
        self._storage = storage
        self._crafts = load_craft_allowlist(craft_allowlist_path)
        self._craft_vocab = load_craft_vocab(craft_allowlist_path)

    async def extract(self, object_keys: list[str], declared_craft_id: str, hint: str) -> dict:
        return await asyncio.to_thread(self._extract_blocking, object_keys, declared_craft_id, hint)

    def _extract_blocking(self, object_keys: list[str], declared_craft_id: str, hint: str) -> dict:
        images = [self._storage.get_bytes(k) for k in object_keys]
        declared_vocab = self._craft_vocab.get(declared_craft_id, {})
        raw = self._vlm.propose_attributes(
            images,
            object_keys,
            self._crafts,
            declared_craft_id,
            hint,
            _vocab_hint(declared_vocab),
            self._craft_vocab,
        )

        # The allowlist is enforced here, not just in the prompt: an id the
        # ontology does not have would break the foreign key on
        # listing_attribute, and a confident wrong craft is worse than an
        # abstention.
        craft = raw.get("craft_id", "")
        if craft not in self._crafts:
            craft = declared_craft_id if declared_craft_id in self._crafts else ""

        craft_vocab = self._craft_vocab.get(craft, {})
        material = _closed_vocab(raw.get("material", ""), craft_vocab.get("materials", []), craft, "material")
        technique = _closed_vocab(raw.get("technique", ""), craft_vocab.get("techniques", []), craft, "technique")
        confidence = raw.get("confidence") or {}

        return {
            "craft_id": (craft, _confidence(confidence, "craft_id") if craft else 0.0),
            "material": (material, _confidence(confidence, "material") if material else 0.0),
            "technique": (technique, _confidence(confidence, "technique") if technique else 0.0),
            "colours": (_strings(raw.get("colours")), _confidence(confidence, "colours")),
            "motifs": (_strings(raw.get("motifs")), _confidence(confidence, "motifs")),
            "dimensions": (raw.get("dimensions") or {}, raw.get("dimensions_confidence", 0.0)),
        }
