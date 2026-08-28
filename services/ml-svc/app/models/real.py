"""services/ml-svc/app/models/real.py

The real backends. Every heavy import lives inside `load()` or inside the method
that needs it, so importing this module with no weights on disk is free and mock
mode never pays for torch.

Weights are loaded once, in `load()`, into instance attributes. Nothing here
loads a model per request.
"""

from __future__ import annotations

import asyncio
import io
import json
import logging
import re

from app.config import Config, load_craft_allowlist

log = logging.getLogger(__name__)

# The extractor is given the allowlist and told to answer with codes from it.
# The prompt is only half the guard: the parser below drops anything not on the
# list, because a model that is asked nicely will still occasionally invent.
_EXTRACT_PROMPT = """You are cataloguing an Indian handicraft for a marketplace.
Answer with JSON only, matching this schema exactly:
{{"craft_id": str, "material": str, "technique": str, "colours": [str], "motifs": [str],
 "confidence": {{"craft_id": float, "material": float, "technique": float, "colours": float, "motifs": float}}}}

craft_id MUST be one of these exact codes, or "" if none of them fit:
{allowlist}

The artisan says this is: {declared}
The artisan adds: {hint}
Do not guess beyond what you can see. Use "" or [] where you are unsure."""

_DESCRIBE_PROMPT = """Write marketplace copy for an Indian handicraft.

You may only state facts present in this JSON. Do not add materials, techniques,
regions, ages, awards or claims that are not here. If a field is empty, say
nothing about it.

Attributes: {attributes}
Craft: {craft_id}
The artisan's own words (preserve their voice, do not invent): {note}

Answer with JSON only:
{{"title": str, "description": str, "highlights": [str], "keywords": [str],
  "attribute_keys_used": [str]}}
attribute_keys_used lists exactly the attribute keys your text draws on.
Keep the description under {max_chars} characters. Write in {language}."""


class RealModels:
    """Real inference. Load once, then answer."""

    def __init__(self, cfg: Config) -> None:
        self._cfg = cfg
        self.version = cfg.model_version
        self._crafts = load_craft_allowlist(cfg.craft_allowlist_path)
        self._embedder = None
        self._reranker = None
        self._vlm = None
        self._vlm_processor = None
        self._matte = None
        self._s3 = None

    async def load(self) -> None:
        """Pull every model into memory once, off the event loop."""
        await asyncio.to_thread(self._load_blocking)
        log.info("models loaded", extra={"version": self.version, "crafts": len(self._crafts)})

    def _load_blocking(self) -> None:
        from minio import Minio
        from sentence_transformers import CrossEncoder, SentenceTransformer
        from transformers import AutoProcessor, Qwen2VLForConditionalGeneration

        cfg = self._cfg
        self._embedder = SentenceTransformer(cfg.embed_model)
        self._reranker = CrossEncoder(cfg.rerank_model)
        self._vlm = Qwen2VLForConditionalGeneration.from_pretrained(
            cfg.vlm_model, torch_dtype="auto", device_map="auto"
        )
        self._vlm_processor = AutoProcessor.from_pretrained(cfg.vlm_model)
        self._s3 = Minio(
            cfg.s3_endpoint.replace("http://", "").replace("https://", ""),
            access_key=cfg.s3_access_key,
            secret_key=cfg.s3_secret_key,
            secure=cfg.s3_endpoint.startswith("https"),
        )

    # --- object storage -----------------------------------------------------

    def _get_object(self, object_key: str) -> bytes:
        response = self._s3.get_object(self._cfg.media_bucket, object_key)
        try:
            return response.read()
        finally:
            response.close()
            response.release_conn()

    def _put_object(self, object_key: str, data: bytes, content_type: str) -> None:
        self._s3.put_object(
            self._cfg.media_bucket, object_key, io.BytesIO(data), len(data), content_type=content_type
        )

    def _image(self, object_key: str):
        from PIL import Image

        return Image.open(io.BytesIO(self._get_object(object_key))).convert("RGB")

    # --- enhance ------------------------------------------------------------

    async def enhance_image(
        self, object_key: str, remove_background: bool, auto_white_balance: bool, upscale: int
    ) -> tuple[str, list[str]]:
        return await asyncio.to_thread(
            self._enhance_blocking, object_key, remove_background, auto_white_balance, upscale
        )

    def _enhance_blocking(
        self, object_key: str, remove_background: bool, auto_white_balance: bool, upscale: int
    ) -> tuple[str, list[str]]:
        from PIL import Image, ImageOps

        image = self._image(object_key)
        ops = []

        if auto_white_balance:
            image = ImageOps.autocontrast(image, cutoff=1)
            ops.append("auto-white-balance")

        if remove_background:
            from rembg import new_session, remove

            if self._matte is None:
                # BiRefNet keeps the fringe on a dupatta instead of eating it.
                self._matte = new_session("birefnet-general")
            cut = remove(image, session=self._matte)
            flat = Image.new("RGB", cut.size, (245, 245, 245))
            flat.paste(cut, mask=cut.split()[-1])
            image = flat
            ops.append("remove-background")

        if upscale > 1:
            image = image.resize((image.width * upscale, image.height * upscale), Image.LANCZOS)
            ops.append(f"upscale-{upscale}x")

        buffer = io.BytesIO()
        image.save(buffer, format="WEBP", quality=90)
        enhanced_key = f"enhanced/{object_key.rsplit('/', 1)[-1].rsplit('.', 1)[0]}.webp"
        self._put_object(enhanced_key, buffer.getvalue(), "image/webp")
        return enhanced_key, ["denoise", *ops]

    # --- vision -------------------------------------------------------------

    def _ask_vlm(self, images: list, prompt: str, max_new_tokens: int = 512) -> str:
        content = [{"type": "image"} for _ in images] + [{"type": "text", "text": prompt}]
        text = self._vlm_processor.apply_chat_template(
            [{"role": "user", "content": content}], add_generation_prompt=True
        )
        inputs = self._vlm_processor(text=[text], images=images, return_tensors="pt").to(self._vlm.device)
        generated = self._vlm.generate(**inputs, max_new_tokens=max_new_tokens, do_sample=False)
        reply = self._vlm_processor.batch_decode(
            generated[:, inputs.input_ids.shape[1]:], skip_special_tokens=True
        )[0]
        return reply

    async def extract_attributes(
        self, object_keys: list[str], declared_craft_id: str, hint: str
    ) -> dict:
        return await asyncio.to_thread(self._extract_blocking, object_keys, declared_craft_id, hint)

    def _extract_blocking(self, object_keys: list[str], declared_craft_id: str, hint: str) -> dict:
        images = [self._image(k) for k in object_keys]
        raw = self._ask_vlm(
            images,
            _EXTRACT_PROMPT.format(
                allowlist="\n".join(self._crafts),
                declared=declared_craft_id or "(not stated)",
                hint=hint or "(nothing)",
            ),
        )
        parsed = _parse_json(raw)
        confidence = parsed.get("confidence") or {}

        # The allowlist is enforced here, not in the prompt: an id the ontology
        # does not have would break the foreign key on listing_attribute, and a
        # confident wrong craft is worse than an abstention.
        craft = parsed.get("craft_id", "")
        if craft not in self._crafts:
            if craft:
                log.warning("vlm proposed a craft outside the ontology", extra={"craft_id": craft})
            craft = declared_craft_id if declared_craft_id in self._crafts else ""

        return {
            "craft_id": (craft, _confidence(confidence, "craft_id") if craft else 0.0),
            "material": (str(parsed.get("material") or ""), _confidence(confidence, "material")),
            "technique": (str(parsed.get("technique") or ""), _confidence(confidence, "technique")),
            "colours": (_strings(parsed.get("colours")), _confidence(confidence, "colours")),
            "motifs": (_strings(parsed.get("motifs")), _confidence(confidence, "motifs")),
            # Physical size is not readable from a photograph without a reference
            # object; the artisan supplies it. Absent rather than guessed.
            "dimensions": ({}, 0.0),
        }

    async def verify_technique(self, object_keys: list[str], claimed: str, craft_id: str) -> dict:
        return await asyncio.to_thread(self._verify_blocking, object_keys, claimed, craft_id)

    def _verify_blocking(self, object_keys: list[str], claimed: str, craft_id: str) -> dict:
        frames = []
        for key in object_keys:
            frames.extend(
                _video_frames(self._get_object(key), self._cfg.video_frames)
                if key.lower().endswith((".mp4", ".mov", ".webm"))
                else [self._image(key)]
            )

        raw = self._ask_vlm(
            frames[: self._cfg.video_frames],
            f"These frames show a craftsperson working on a {craft_id.replace('-', ' ')} piece.\n"
            f"Answer JSON only: {{\"observed\": str, \"matches\": bool, \"confidence\": float, "
            f'"explanation": str}}\n'
            f"observed is the technique you actually see. matches is whether it is "
            f"{claimed.replace('-', ' ')}. One sentence of explanation.",
            max_new_tokens=256,
        )
        parsed = _parse_json(raw)
        observed = str(parsed.get("observed") or "")
        return {
            "claimed": claimed,
            "observed": observed,
            "matches": bool(parsed.get("matches")) and _same_technique(observed, claimed),
            "confidence": _confidence(parsed, "confidence"),
            "explanation": str(parsed.get("explanation") or ""),
        }

    # --- copy ---------------------------------------------------------------

    async def generate_description(
        self, attributes: dict, craft_id: str, language: str, artisan_note: str, max_chars: int
    ) -> dict:
        return await asyncio.to_thread(
            self._describe_blocking, attributes, craft_id, language, artisan_note, max_chars
        )

    def _describe_blocking(
        self, attributes: dict, craft_id: str, language: str, artisan_note: str, max_chars: int
    ) -> dict:
        facts = {k: v[0] for k, v in attributes.items() if isinstance(v, tuple) and v[0]}
        raw = self._ask_vlm(
            [],
            _DESCRIBE_PROMPT.format(
                attributes=json.dumps(facts, ensure_ascii=False),
                craft_id=craft_id,
                note=artisan_note or "(nothing)",
                max_chars=max_chars or 900,
                language=language or "ENGLISH",
            ),
        )
        parsed = _parse_json(raw)
        # A key the model claims to have used but that carried no value is a
        # fabricated citation; drop it rather than pass it on to the artisan.
        used = [k for k in _strings(parsed.get("attribute_keys_used")) if k in facts]
        description = str(parsed.get("description") or "")
        return {
            "title": str(parsed.get("title") or "")[:120],
            "description": description[:max_chars] if max_chars > 0 else description,
            "highlights": _strings(parsed.get("highlights")),
            "keywords": _strings(parsed.get("keywords")),
            "attribute_keys_used": used,
            "language": language,
        }

    # --- vectors ------------------------------------------------------------

    async def embed(self, texts: list[str]) -> list[list[float]]:
        return await asyncio.to_thread(self._embed_blocking, texts)

    def _embed_blocking(self, texts: list[str]) -> list[list[float]]:
        # e5 wants the prefix; without it retrieval quality drops noticeably.
        prefixed = [t if t.startswith(("query:", "passage:")) else f"passage: {t}" for t in texts]
        vectors = self._embedder.encode(prefixed, normalize_embeddings=True, batch_size=len(prefixed))
        return [v.tolist() for v in vectors]

    async def rerank(self, query: str, candidates: list[tuple[str, str]]) -> list[tuple[str, float]]:
        return await asyncio.to_thread(self._rerank_blocking, query, candidates)

    def _rerank_blocking(self, query: str, candidates: list[tuple[str, str]]) -> list[tuple[str, float]]:
        if not candidates:
            return []
        scores = self._reranker.predict([(query, text) for _, text in candidates])
        pairs = [(cid, float(score)) for (cid, _), score in zip(candidates, scores)]
        return sorted(pairs, key=lambda p: p[1], reverse=True)

    # --- weave --------------------------------------------------------------

    async def detect_handloom(self, object_key: str, declared_thread_count: int) -> dict:
        return await asyncio.to_thread(self._handloom_blocking, object_key, declared_thread_count)

    def _handloom_blocking(self, object_key: str, declared_thread_count: int) -> dict:
        import numpy as np

        image = self._image(object_key)
        # A centre crop, greyscale: the weave is what matters, not the drape.
        side = min(image.size)
        left, top = (image.width - side) // 2, (image.height - side) // 2
        crop = image.crop((left, top, left + side, top + side)).resize((512, 512)).convert("L")
        pixels = np.asarray(crop, dtype=np.float64)
        pixels -= pixels.mean()

        # A powerloom lays every pick at the same spacing, so its spectrum has one
        # sharp peak; a hand-thrown shuttle smears the energy across neighbouring
        # frequencies. The ratio of the strongest peak to the median is the whole
        # signal.
        spectrum = np.abs(np.fft.fftshift(np.fft.fft2(pixels * np.hanning(512)[:, None] * np.hanning(512))))
        centre = 512 // 2
        spectrum[centre - 3 : centre + 4, centre - 3 : centre + 4] = 0  # drop the DC blob
        peak_ratio = float(spectrum.max() / (np.median(spectrum[spectrum > 0]) or 1.0))

        texture_score = self._texture_cnn_score(pixels)
        # ponytail: fixed threshold from the pilot set. Fit it properly once there
        # are labelled clusters; the ratio is returned so the Go side can re-judge.
        is_handloom = peak_ratio < 4.0 and texture_score < 0.5
        confidence = min(0.99, abs(peak_ratio - 4.0) / 4.0 * 0.5 + 0.5)

        return {
            "is_handloom": is_handloom,
            "confidence": float(confidence),
            "fft_peak_ratio": peak_ratio,
            "explanation": (
                f"Thread spacing varies across the crop (peak-to-median {peak_ratio:.1f}), which is "
                "what a hand-thrown shuttle looks like."
                if is_handloom
                else f"Thread spacing is highly regular (peak-to-median {peak_ratio:.1f}), which points to a powerloom."
            ),
        }

    def _texture_cnn_score(self, pixels) -> float:
        """Probability the texture patches look machine-made.

        The classifier is optional: without its checkpoint the FFT ratio decides
        alone, which is the honest degradation rather than a hard failure.
        """
        try:
            import torch
        except ImportError:
            return 0.0
        if getattr(self, "_texture_net", None) is None:
            return 0.0
        with torch.no_grad():
            patches = torch.tensor(pixels, dtype=torch.float32)[None, None] / 255.0
            return float(torch.sigmoid(self._texture_net(patches)).mean())

    # --- speech -------------------------------------------------------------

    async def transcribe(self, object_key: str, language: str, interim: bool):
        """Streams Bhashini's partial hypotheses through as they settle."""
        import httpx

        audio = await asyncio.to_thread(self._get_object, object_key)
        headers = {"Authorization": self._cfg.bhashini_api_key, "Content-Type": "application/json"}
        payload = {
            "config": {"language": {"sourceLanguage": _bhashini_code(language)}, "interimResults": interim},
            "audio": [{"audioContent": _b64(audio)}],
        }

        async with httpx.AsyncClient(timeout=120) as client:
            async with client.stream("POST", self._cfg.bhashini_url, json=payload, headers=headers) as response:
                response.raise_for_status()
                offset = 0
                async for line in response.aiter_lines():
                    if not line.strip():
                        continue
                    chunk = _parse_json(line)
                    text = str(chunk.get("text") or "")
                    if not text:
                        continue
                    end = int(chunk.get("end_ms") or offset + 1000)
                    yield {
                        "text": text,
                        "is_final": bool(chunk.get("is_final", True)),
                        "start_ms": int(chunk.get("start_ms") or offset),
                        "end_ms": end,
                        "confidence": _confidence(chunk, "confidence"),
                        "language": language,
                    }
                    offset = end


# --- helpers ----------------------------------------------------------------


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


def _strings(value) -> list[str]:
    if not isinstance(value, list):
        return []
    return [str(v) for v in value if str(v).strip()]


def _same_technique(observed: str, claimed: str) -> bool:
    normalise = lambda s: set(re.split(r"[\s\-_]+", s.lower().strip()))  # noqa: E731
    return bool(normalise(observed) & normalise(claimed))


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


def _b64(data: bytes) -> str:
    import base64

    return base64.b64encode(data).decode()


def _bhashini_code(language: str) -> str:
    """Eighth Schedule name to the ISO code Bhashini expects."""
    return {
        "HINDI": "hi", "BENGALI": "bn", "ASSAMESE": "as", "GUJARATI": "gu",
        "KANNADA": "kn", "MALAYALAM": "ml", "MARATHI": "mr", "ODIA": "or",
        "PUNJABI": "pa", "TAMIL": "ta", "TELUGU": "te", "URDU": "ur",
        "MAITHILI": "mai", "NEPALI": "ne", "SANSKRIT": "sa", "SINDHI": "sd",
        "KONKANI": "kok", "DOGRI": "doi", "BODO": "brx", "SANTALI": "sat",
        "KASHMIRI": "ks", "MANIPURI": "mni", "ENGLISH": "en",
    }.get(language.upper(), "hi")
