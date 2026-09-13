"""services/ml-svc/app/models/translation/real.py

Wraps AI4Bharat's IndicTrans2. Heavy imports (torch, transformers,
IndicTransToolkit) live inside `_ensure_loaded`, never at module scope --
mock mode must never need them, matching every other component in this
service (see services/ml-svc/CLAUDE.md).

Standing constraint: this file is never installed or run in this session --
see services/ml-svc/CLAUDE.md's "never run installs or downloads that pull
heavy files/models". A real deployment adds `torch`, `transformers`, and
`IndicTransToolkit` to this service's `real` extra, sets
`ML_SVC_TRANSLATION_MODEL` to a checkpoint (the en-indic 1B distilled
checkpoint is the default), and this is the code path that runs.

IndicTrans2 identifies languages by FLORES-200-style codes, not by the
`language_code` DB enum's plain names. The mapping below covers this
project's supported languages (see `pkg/config`'s `PIPELINE_BUYER_LANGUAGES`
and `packages/i18n/src/locales.ts` for the matching frontend list) --
Bodo/Dogri/Konkani/Maithili/Kashmiri/Sindhi codes are IndicTrans2's
documented extension beyond the core FLORES-200 set and should be verified
against the actual checkpoint's `config.json` language list before this
runs for real, since a checkpoint swap could use different codes.
"""

from __future__ import annotations

from app.models.translation import TranslationConfig

_LANGUAGE_TO_FLORES_CODE = {
    "ENGLISH": "eng_Latn",
    "ASSAMESE": "asm_Beng",
    "BENGALI": "ben_Beng",
    "BODO": "brx_Deva",
    "DOGRI": "doi_Deva",
    "GUJARATI": "guj_Gujr",
    "HINDI": "hin_Deva",
    "KANNADA": "kan_Knda",
    "KASHMIRI": "kas_Arab",
    "KONKANI": "gom_Deva",
    "MAITHILI": "mai_Deva",
    "MALAYALAM": "mal_Mlym",
    "MARATHI": "mar_Deva",
    "NEPALI": "npi_Deva",
    "ODIA": "ory_Orya",
    "PUNJABI": "pan_Guru",
    "SANSKRIT": "san_Deva",
    "SINDHI": "snd_Arab",
    "TAMIL": "tam_Taml",
    "TELUGU": "tel_Telu",
    "URDU": "urd_Arab",
}


class RealTranslationModel:
    def __init__(self, cfg: TranslationConfig) -> None:
        self._cfg = cfg
        self._tokenizer = None
        self._model = None
        self._processor = None

    def translate(
        self,
        title: str,
        description: str,
        highlights: list[str],
        source_language: str,
        target_language: str,
    ) -> dict:
        src = _LANGUAGE_TO_FLORES_CODE.get(source_language, "eng_Latn")
        tgt = _LANGUAGE_TO_FLORES_CODE.get(target_language)
        if tgt is None or tgt == src:
            # Unknown or no-op target: hand the text back rather than guess.
            return {"title": title, "description": description, "highlights": list(highlights)}

        self._ensure_loaded()
        texts = [title, description, *highlights]
        translated = self._translate_batch(texts, src, tgt)
        return {
            "title": translated[0],
            "description": translated[1],
            "highlights": translated[2:],
        }

    def _ensure_loaded(self) -> None:
        if self._model is not None:
            return
        from transformers import AutoModelForSeq2SeqLM, AutoTokenizer
        from IndicTransToolkit.processor import IndicProcessor

        self._tokenizer = AutoTokenizer.from_pretrained(self._cfg.model, trust_remote_code=True)
        self._model = AutoModelForSeq2SeqLM.from_pretrained(self._cfg.model, trust_remote_code=True)
        self._model.eval()
        self._processor = IndicProcessor(inference=True)

    def _translate_batch(self, texts: list[str], src: str, tgt: str) -> list[str]:
        import torch

        batch = self._processor.preprocess_batch(texts, src_lang=src, tgt_lang=tgt)
        inputs = self._tokenizer(batch, padding=True, truncation=True, return_tensors="pt")
        with torch.no_grad():
            generated = self._model.generate(**inputs, max_length=256, num_beams=5)
        decoded = self._tokenizer.batch_decode(generated, skip_special_tokens=True)
        return self._processor.postprocess_batch(decoded, lang=tgt)
