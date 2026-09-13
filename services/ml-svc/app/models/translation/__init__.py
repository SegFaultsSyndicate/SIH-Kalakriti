"""services/ml-svc/app/models/translation/

The machine-translation component. Backs the Translate RPC, which
`core-svc`'s pipeline calls once per (listing, target language) to fan a
published listing's copy out to every buyer language.

Do-not-translate placeholders (`{{dnt<N>}}`, see core-svc's
`pipeline.go`'s `mask`/`unmask`) are the calling Go pipeline's concern, not
this component's: masking happens before the request ever reaches ml-svc,
and unmasking happens after the response comes back. Every backend here
must treat the braces as opaque text and pass them through unchanged --
they are not a component-level guarantee to implement, just a property of
the text this component is never told about.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class TranslationModel(Protocol):
    def translate(
        self,
        title: str,
        description: str,
        highlights: list[str],
        source_language: str,
        target_language: str,
    ) -> dict: ...


@dataclass(frozen=True)
class TranslationConfig:
    """AI4Bharat's IndicTrans2 is purpose-built for all 22 scheduled Indian
    languages (a superset of the 20 this project supports after dropping
    Manipuri/Santali), open-weight, and self-hostable alongside the
    hf-cache volume the other components already share -- no per-call
    cost, no new external dependency."""

    model: str = "ai4bharat/indictrans2-en-indic-1B"

    @classmethod
    def from_env(cls) -> "TranslationConfig":
        return cls(model=os.getenv("ML_SVC_TRANSLATION_MODEL", cls.model))


def build(cfg: Config) -> TranslationModel:
    if cfg.mock_mode:
        from app.models.translation.mock import MockTranslationModel

        return MockTranslationModel()

    from app.models.translation.real import RealTranslationModel

    return RealTranslationModel(TranslationConfig.from_env())
