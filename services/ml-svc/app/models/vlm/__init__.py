"""services/ml-svc/app/models/vlm/

The vision-language-model component. Backs three RPCs (extract_attributes,
verify_technique, generate_description's polish pass) -- one loaded model,
three thin methods, matching how `RealModels` already shared one VLM across
all three before this refactor.

Every method returns a *raw* proposal: allowlist/closed-vocabulary
enforcement, the video-vs-image decision aside (that's genuinely
real-model-specific, see `real.py`), and the description grounding check are
all pure, model-agnostic logic and live in the calling `features/*.py`
module instead -- a real component's hallucination and a mock component's
fabrication both flow through the exact same guard.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class VisionLanguageModel(Protocol):
    def propose_attributes(
        self,
        images: list[bytes],
        object_keys: list[str],
        allowlist: list[str],
        declared_craft_id: str,
        hint: str,
        vocab_hint: str,
        craft_vocab: dict[str, dict[str, list[str]]],
    ) -> dict: ...

    def observe_technique(
        self,
        media: list[bytes],
        object_keys: list[str],
        claimed: str,
        craft_id: str,
        video_frames: int,
    ) -> dict: ...

    def polish(self, template: str, facts: dict, language: str, max_chars: int) -> str: ...


@dataclass(frozen=True)
class VLMConfig:
    """One repo id serves all three RPCs above -- see the three canonical env
    var names below, which all default to it. A deployment that wants a
    different model per RPC (e.g. a bigger model just for description polish)
    sets the specific var; nothing here forces them to be the same repo.

    `backend` picks which real implementation actually runs the model:
    "transformers" (`real.py`, in-process HF/bitsandbytes -- the only path
    that ever ran on a small/no-GPU box) or "llamacpp" (`llamacpp.py`, an
    HTTP client to a separate `llama-server` sidecar). See
    `services/ml-svc/CLAUDE.md` for why both are kept rather than one
    replacing the other."""

    attribute_extraction_model: str = "Qwen/Qwen2.5-VL-3B-Instruct"
    technique_verification_model: str = "Qwen/Qwen2.5-VL-3B-Instruct"
    description_model: str = "Qwen/Qwen2.5-VL-3B-Instruct"
    # How many frames VerifyTechnique samples from a video clip -- a
    # processing knob for that one feature, not a model identity, but it
    # lives here because it is only ever read alongside this component.
    video_frames: int = 8
    backend: str = "transformers"
    llamacpp_base_url: str = "http://llama-server:8080"

    @classmethod
    def from_env(cls) -> "VLMConfig":
        default = "Qwen/Qwen2.5-VL-3B-Instruct"
        return cls(
            attribute_extraction_model=os.getenv("ML_SVC_IMAGE_ATTRIBUTE_EXTRACTION_MODEL", default),
            technique_verification_model=os.getenv("ML_SVC_TECHNIQUE_VERIFICATION_MODEL", default),
            description_model=os.getenv("ML_SVC_IMAGE_DESCRIPTION_MODEL", default),
            video_frames=int(os.getenv("ML_SVC_VIDEO_FRAMES", "8")),
            backend=os.getenv("ML_SVC_VLM_BACKEND", "transformers"),
            llamacpp_base_url=os.getenv("ML_SVC_VLM_LLAMACPP_URL", "http://llama-server:8080"),
        )


def build(cfg: Config) -> VisionLanguageModel:
    if cfg.mock_mode:
        from app.models.vlm.mock import MockVisionLanguageModel

        return MockVisionLanguageModel()

    vlm_cfg = VLMConfig.from_env()
    if vlm_cfg.backend == "llamacpp":
        from app.models.vlm.llamacpp import LlamaCppVisionLanguageModel

        return LlamaCppVisionLanguageModel(vlm_cfg)

    from app.models.vlm.real import RealVisionLanguageModel

    return RealVisionLanguageModel(vlm_cfg)
