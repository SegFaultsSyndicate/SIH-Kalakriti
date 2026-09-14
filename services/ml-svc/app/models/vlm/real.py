"""services/ml-svc/app/models/vlm/real.py

Qwen2.5-VL-backed. `torch`/`transformers`/`PIL`/`cv2` are imported here, not at
module scope in `__init__.py`, so mock mode never needs them installed.

One process loads exactly one VLM checkpoint today, from
`VLMConfig.attribute_extraction_model` -- the three canonical env vars exist
so a deployment can *name* a different model per RPC in config, but actually
running three different checkpoints concurrently would mean loading three
separate model instances, which this component does not do yet. If
`technique_verification_model`/`description_model` are set to something
different, a warning is logged at load time so that gap is visible rather
than silently ignored.
"""

from __future__ import annotations

import json
import logging
import re

from app.models.vlm import VLMConfig

log = logging.getLogger(__name__)

_EXTRACT_PROMPT = """You are cataloguing an Indian handicraft for a marketplace.
Answer with JSON only, matching this schema exactly:
{{"craft_id": str, "material": str, "technique": str, "colours": [str], "motifs": [str],
 "confidence": {{"craft_id": float, "material": float, "technique": float, "colours": float, "motifs": float}}}}

craft_id MUST be one of these exact codes, or "" if none of them fit:
{allowlist}

The artisan says this is: {declared}
The artisan adds: {hint}{vocab_hint}
Do not guess beyond what you can see. Use "" or [] where you are unsure."""

_POLISH_PROMPT = """Rewrite this marketplace product description with a warmer,
more persuasive tone. State only the facts already in it — do not add any
material, technique, colour, motif, region, age, award or other claim that is
not already present in the text below.

{template}

Write in {language}, under {max_chars} characters. Reply with the rewritten
description only: no JSON, no preamble, no quotation marks."""


class RealVisionLanguageModel:
    def __init__(self, cfg: VLMConfig) -> None:
        if len({cfg.attribute_extraction_model, cfg.technique_verification_model, cfg.description_model}) > 1:
            log.warning(
                "vlm component only loads one checkpoint but the three "
                "VLM-backed settings name different repos; using %r for all three",
                cfg.attribute_extraction_model,
            )
        self._repo_id = cfg.attribute_extraction_model
        self._model = None
        self._processor = None

    def _load(self) -> None:
        if self._model is not None:
            return

        from transformers import AutoProcessor, Qwen2_5_VLForConditionalGeneration

        from app.model_loading import local_files_only_kwargs, resolve_device

        device, dtype = resolve_device()
        # device_map="auto" lets accelerate plan across multiple accelerators --
        # a fit for a genuinely multi-GPU host, not this component, which loads
        # exactly one checkpoint on one process (see this module's docstring).
        # On a CPU-only host it can still decide to offload layers to disk under
        # memory pressure, which hits a known accelerate bug for tied-weight
        # models (KeyError: 'cpu' in the tied_params_map hook cleanup -- Qwen2.5-VL
        # ties embed_tokens/lm_head). Passing the resolved device directly avoids
        # that path entirely.
        self._model = Qwen2_5_VLForConditionalGeneration.from_pretrained(
            self._repo_id, torch_dtype=dtype, device_map=device, **local_files_only_kwargs(self._repo_id)
        )
        self._processor = AutoProcessor.from_pretrained(
            self._repo_id, **local_files_only_kwargs(self._repo_id)
        )

    def _ask(self, images: list, prompt: str, max_new_tokens: int = 512) -> str:
        self._load()
        content = [{"type": "image"} for _ in images] + [{"type": "text", "text": prompt}]
        text = self._processor.apply_chat_template(
            [{"role": "user", "content": content}], add_generation_prompt=True
        )
        # An empty list (the text-only `polish()` call) must become None, not
        # `[]` -- the processor treats `images=[]` as "process zero images"
        # rather than "no images", still driving the vision tower's
        # rot_pos_emb into torch.cat() on an empty tensor list.
        inputs = self._processor(text=[text], images=images or None, return_tensors="pt").to(self._model.device)
        generated = self._model.generate(**inputs, max_new_tokens=max_new_tokens, do_sample=False)
        return self._processor.batch_decode(
            generated[:, inputs.input_ids.shape[1]:], skip_special_tokens=True
        )[0]

    def propose_attributes(
        self,
        images: list[bytes],
        object_keys: list[str],
        allowlist: list[str],
        declared_craft_id: str,
        hint: str,
        vocab_hint: str,
        craft_vocab: dict[str, dict[str, list[str]]],
    ) -> dict:
        from PIL import Image
        import io

        decoded = [Image.open(io.BytesIO(data)).convert("RGB") for data in images]
        raw = self._ask(
            decoded,
            _EXTRACT_PROMPT.format(
                allowlist="\n".join(allowlist),
                declared=declared_craft_id or "(not stated)",
                hint=hint or "(nothing)",
                vocab_hint=vocab_hint,
            ),
        )
        parsed = _parse_json(raw)
        return {
            "craft_id": str(parsed.get("craft_id") or ""),
            "material": str(parsed.get("material") or ""),
            "technique": str(parsed.get("technique") or ""),
            "colours": parsed.get("colours"),
            "motifs": parsed.get("motifs"),
            "confidence": parsed.get("confidence") or {},
            # Physical size is not readable from a photograph without a
            # reference object; the artisan supplies it. Absent rather than
            # guessed.
            "dimensions": None,
            "dimensions_confidence": 0.0,
        }

    def observe_technique(
        self,
        media: list[bytes],
        object_keys: list[str],
        claimed: str,
        craft_id: str,
        video_frames: int,
    ) -> dict:
        frames = []
        for data, key in zip(media, object_keys):
            if key.lower().endswith((".mp4", ".mov", ".webm")):
                frames.extend(_video_frames(data, video_frames))
            else:
                from PIL import Image
                import io

                frames.append(Image.open(io.BytesIO(data)).convert("RGB"))

        raw = self._ask(
            frames[:video_frames],
            f"These frames show a craftsperson working on a {craft_id.replace('-', ' ')} piece.\n"
            f"Answer JSON only: {{\"observed\": str, \"matches\": bool, \"confidence\": float, "
            f'"explanation": str}}\n'
            f"observed is the technique you actually see. matches is whether it is "
            f"{claimed.replace('-', ' ')}. One sentence of explanation.",
            max_new_tokens=256,
        )
        parsed = _parse_json(raw)
        return {
            "observed": str(parsed.get("observed") or ""),
            "matches": bool(parsed.get("matches")),
            "confidence": _confidence(parsed, "confidence"),
            "explanation": str(parsed.get("explanation") or ""),
        }

    def polish(self, template: str, facts: dict, language: str, max_chars: int) -> str:
        raw = self._ask(
            [],
            _POLISH_PROMPT.format(template=template, language=language or "ENGLISH", max_chars=max_chars or 900),
            max_new_tokens=256,
        )
        return raw.strip()


def _parse_json(raw: str) -> dict:
    """Pull the JSON object out of a model reply, fenced or not."""
    match = re.search(r"\{.*\}", raw, re.S)
    if not match:
        log.warning("model reply carried no json", extra={"reply": raw[:200]})
        return {}
    try:
        return json.loads(match.group(0))
    except json.JSONDecodeError:
        log.warning("model reply was not valid json", extra={"reply": raw[:200]})
        return {}


def _confidence(source: dict, key: str) -> float:
    try:
        return max(0.0, min(1.0, float(source.get(key, 0.0))))
    except (TypeError, ValueError):
        return 0.0


def _video_frames(data: bytes, count: int) -> list:
    """N frames spread evenly across a clip, as PIL images."""
    import tempfile

    import cv2
    from PIL import Image

    with tempfile.NamedTemporaryFile(suffix=".mp4") as handle:
        handle.write(data)
        handle.flush()
        capture = cv2.VideoCapture(handle.name)
        try:
            total = int(capture.get(cv2.CAP_PROP_FRAME_COUNT)) or 1
            frames = []
            for i in range(count):
                capture.set(cv2.CAP_PROP_POS_FRAMES, int(i * total / count))
                ok, frame = capture.read()
                if ok:
                    frames.append(Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)))
            return frames
        finally:
            capture.release()
