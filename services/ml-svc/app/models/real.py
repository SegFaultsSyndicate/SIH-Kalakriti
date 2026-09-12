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

from app.config import Config, load_craft_allowlist, load_craft_vocab

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
The artisan adds: {hint}{vocab_hint}
Do not guess beyond what you can see. Use "" or [] where you are unsure."""

_POLISH_PROMPT = """Rewrite this marketplace product description with a warmer,
more persuasive tone. State only the facts already in it — do not add any
material, technique, colour, motif, region, age, award or other claim that is
not already present in the text below.

{template}

Write in {language}, under {max_chars} characters. Reply with the rewritten
description only: no JSON, no preamble, no quotation marks."""


class RealModels:
    """Real inference. Load once, then answer."""

    def __init__(self, cfg: Config) -> None:
        self._cfg = cfg
        self.version = cfg.model_version
        self._crafts = load_craft_allowlist(cfg.craft_allowlist_path)
        self._craft_vocab = load_craft_vocab(cfg.craft_allowlist_path)
        self._embedder = None
        self._reranker = None
        self._vlm = None
        self._vlm_processor = None
        self._matte = None
        self._s3 = None
        # Lazily loaded on first correct_lighting=True request, not in
        # `load()`: unlike the always-available rembg/birefnet-general
        # session, there is no bundled Zero-DCE++ checkpoint, so an install
        # with none configured must not fail startup over an optional op.
        self._zero_dce_net = None
        self._zero_dce_load_failed = False

    async def load(self) -> None:
        """Pull every model into memory once, off the event loop."""
        await asyncio.to_thread(self._load_blocking)
        log.info("models loaded", extra={"version": self.version, "crafts": len(self._crafts)})

    def _load_blocking(self) -> None:
        from minio import Minio
        from sentence_transformers import CrossEncoder, SentenceTransformer
        from transformers import AutoProcessor, Qwen2VLForConditionalGeneration

        from app.model_loading import local_files_only_kwargs, resolve_device

        cfg = self._cfg
        device, dtype = resolve_device()
        # local_files_only_kwargs avoids a Hub network round-trip on every
        # restart once a model is actually cached: these three repo ids are
        # pinned, not floating, so there is never a newer revision to check for.
        self._embedder = SentenceTransformer(
            cfg.embed_model, device=device, **local_files_only_kwargs(cfg.embed_model)
        )
        self._reranker = CrossEncoder(
            cfg.rerank_model, device=device, **local_files_only_kwargs(cfg.rerank_model)
        )
        self._vlm = Qwen2VLForConditionalGeneration.from_pretrained(
            cfg.vlm_model, torch_dtype=dtype, device_map="auto", **local_files_only_kwargs(cfg.vlm_model)
        )
        self._vlm_processor = AutoProcessor.from_pretrained(
            cfg.vlm_model, **local_files_only_kwargs(cfg.vlm_model)
        )
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
        self,
        object_key: str,
        remove_background: bool,
        auto_white_balance: bool,
        upscale: int,
        correct_lighting: bool = False,
    ) -> tuple[str, list[str]]:
        return await asyncio.to_thread(
            self._enhance_blocking,
            object_key,
            remove_background,
            auto_white_balance,
            upscale,
            correct_lighting,
        )

    def _enhance_blocking(
        self,
        object_key: str,
        remove_background: bool,
        auto_white_balance: bool,
        upscale: int,
        correct_lighting: bool = False,
    ) -> tuple[str, list[str]]:
        from PIL import Image, ImageOps

        image = self._image(object_key)
        ops = []

        if auto_white_balance:
            image = ImageOps.autocontrast(image, cutoff=1)
            ops.append("auto-white-balance")

        if correct_lighting:
            net = self._get_zero_dce_net()
            if net is not None:
                image = _apply_zero_dce(image, net)
                ops.append("correct-lighting")
            else:
                log.warning(
                    "correct_lighting requested but no Zero-DCE++ checkpoint is "
                    "configured (ML_SVC_ZERO_DCE_CHECKPOINT); skipping"
                )

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

    def _get_zero_dce_net(self):
        """Lazily loads the Zero-DCE++ curve-estimation net, once.

        Optional, the same honest-degradation pattern as
        `_texture_cnn_score`'s missing checkpoint: Zero-DCE++ has no pip
        package or HF repo id, only a raw `.pth` from the paper's GitHub
        release (see `app/models/zero_dce.py`'s module docstring), so an
        install with `ML_SVC_ZERO_DCE_CHECKPOINT` unset must not fail
        startup, or even this call, over an optional enhancement op -- it
        just returns None and the caller skips the op.
        """
        if self._zero_dce_net is not None or self._zero_dce_load_failed:
            return self._zero_dce_net

        path = self._cfg.zero_dce_checkpoint_path
        if not path:
            self._zero_dce_load_failed = True
            return None

        try:
            import torch

            from app.models.zero_dce import ZeroDCEPPNet

            net = ZeroDCEPPNet()
            state_dict = torch.load(path, map_location="cpu")
            net.load_state_dict(state_dict)
            net.eval()
        except Exception:
            log.exception(
                "failed to load Zero-DCE++ checkpoint at %r; correct_lighting will be a no-op",
                path,
            )
            self._zero_dce_load_failed = True
            return None

        self._zero_dce_net = net
        return net

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
                vocab_hint=_vocab_hint(self._craft_vocab.get(declared_craft_id, {})),
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

        # Once the craft is known, material/technique are constrained to *that
        # craft's own* seeded vocabulary — the same closed-vocabulary discipline
        # the craft_id check above already applies, just one level down. A model
        # asked nicely to stay on-list still occasionally invents ("silk" for a
        # terracotta piece); this is the structural guard, not a prompting hope.
        craft_vocab = self._craft_vocab.get(craft, {})
        material = _closed_vocab(str(parsed.get("material") or ""), craft_vocab.get("materials", []), craft, "material")
        technique = _closed_vocab(str(parsed.get("technique") or ""), craft_vocab.get("techniques", []), craft, "technique")

        return {
            "craft_id": (craft, _confidence(confidence, "craft_id") if craft else 0.0),
            "material": (material, _confidence(confidence, "material") if material else 0.0),
            "technique": (technique, _confidence(confidence, "technique") if technique else 0.0),
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
        # Template-first, LLM-polish-second (ported from kalakriti-ml-svc's
        # GenerateDescription): a deterministic sentence built only from
        # `facts` is the structural grounding guarantee — attribute_keys_used
        # is exactly the set of keys the template drew from, not the model's
        # say-so about what it used. The VLM only gets to polish tone from
        # there, and only keeps its rewrite if `_is_grounded` confirms every
        # cited value survived.
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
            raw = self._ask_vlm(
                [],
                _POLISH_PROMPT.format(
                    template=template, language=language or "ENGLISH", max_chars=max_chars or 900
                ),
                max_new_tokens=256,
            )
        except Exception:
            log.exception("polish pass failed, falling back to the template sentence")
            return template

        rewrite = raw.strip()
        if not _is_grounded(rewrite, facts):
            log.warning("polish rewrite dropped or altered a cited attribute value; using the template")
            return template
        return rewrite

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


def _template_sentence(facts: dict, craft_id: str) -> str:
    """A deterministic, factual sentence — the description's grounded core.

    One clause per present attribute, same discipline as
    kalakriti-ml-svc's `_template_sentence`: nothing here is asserted unless
    `facts` actually carries it.
    """
    craft_name = craft_id.replace("-", " ").title() if craft_id else "handcrafted piece"
    clause = f"A {craft_name}"
    if facts.get("material"):
        clause += f" made of {facts['material']}"
    if facts.get("technique"):
        clause += f" using {facts['technique'].replace('-', ' ')}"
    sentences = [clause + "."]

    colours = facts.get("colours") or []
    if colours:
        sentences.append(f"Finished in {' and '.join(colours)}.")
    motifs = facts.get("motifs") or []
    if motifs:
        sentences.append(f"Carries {' and '.join(motifs)} motifs.")
    sentences.append("Each piece is finished by hand, so no two are identical.")
    return " ".join(sentences)


def _template_title(facts: dict, craft_id: str) -> str:
    craft_name = craft_id.replace("-", " ").title() if craft_id else "Handcrafted Piece"
    colour = next(iter(facts.get("colours") or []), None)
    parts = [p.title() for p in (colour, facts.get("material")) if p]
    parts.append(craft_name)
    return " ".join(parts)


def _template_highlights(facts: dict) -> list[str]:
    highlights = ["Hand-finished by a single artisan"]
    if facts.get("material"):
        highlights.append(f"Made from {facts['material']}")
    if facts.get("technique"):
        highlights.append(f"Crafted using {facts['technique'].replace('-', ' ')}")
    return highlights


def _template_keywords(facts: dict, craft_id: str) -> list[str]:
    keywords = [craft_id] if craft_id else []
    keywords += [facts[k] for k in ("material", "technique") if facts.get(k)]
    keywords += facts.get("colours") or []
    keywords += facts.get("motifs") or []
    return keywords


def _is_grounded(rewrite: str, facts: dict) -> bool:
    """True when every cited fact value still appears in `rewrite`,
    case-insensitively — the same substring check kalakriti-ml-svc used, with
    the same known blind spot: it confirms values aren't dropped or swapped,
    not that no *other*, unrelated claim was invented around them. Good
    enough to catch a rewrite that drops or substitutes a material/colour;
    §5.B's artisan-review-before-publish step is the actual safety net for
    anything this substring check can't see.
    """
    if len(rewrite) < 10:
        return False
    lowered = rewrite.lower()
    for value in facts.values():
        for v in (value if isinstance(value, list) else [value]):
            if isinstance(v, str) and v and v.lower() not in lowered:
                return False
    return True


def _vocab_hint(vocab: dict) -> str:
    """A prompt fragment naming a known craft's material/technique vocabulary.

    Only ever a hint, never the enforcement — `_closed_vocab` below is what
    actually guarantees the answer stays on-list. Empty when the craft isn't
    declared yet or the seed CSV has no vocabulary columns for it.
    """
    materials = vocab.get("materials") or []
    techniques = vocab.get("techniques") or []
    if not materials and not techniques:
        return ""
    lines = []
    if materials:
        lines.append(f"materials known for this craft: {', '.join(materials)}")
    if techniques:
        lines.append(f"techniques known for this craft: {', '.join(techniques)}")
    return "\n" + "\n".join(lines)


def _closed_vocab(value: str, vocab: list[str], craft: str, field: str) -> str:
    """Keeps `value` only if it matches an entry of `vocab`, case-insensitively.

    A craft with no recorded vocabulary (`vocab == []`) means the seed CSV had
    no techniques/materials columns for it, not that anything goes — absent
    vocabulary data isn't a constraint, the same rule `load_craft_allowlist`
    already follows for a plain-code-list CSV. So an empty vocab passes
    `value` through unchanged rather than rejecting it.
    """
    if not value or not vocab:
        return value
    by_lower = {v.lower(): v for v in vocab}
    match = by_lower.get(value.lower())
    if match is None:
        log.warning(
            "vlm proposed a %s outside the craft's vocabulary",
            field,
            extra={"craft_id": craft, field: value},
        )
        return ""
    return match


def _same_technique(observed: str, claimed: str) -> bool:
    normalise = lambda s: set(re.split(r"[\s\-_]+", s.lower().strip()))  # noqa: E731
    return bool(normalise(observed) & normalise(claimed))


def _apply_zero_dce(image, net):
    """Runs one Zero-DCE++ forward pass over `image` (a PIL RGB Image) and
    returns the brightened result as a new PIL Image.

    Full resolution, no batching: this is one photo per call, same as every
    other per-image op in `_enhance_blocking`. See `app/models/zero_dce.py`'s
    module docstring for why this deliberately skips upstream's own
    downsample-then-upsample speed trick.
    """
    import numpy as np
    import torch
    from PIL import Image

    array = np.asarray(image, dtype=np.float32) / 255.0
    tensor = torch.from_numpy(array).permute(2, 0, 1).unsqueeze(0)
    with torch.no_grad():
        enhanced = net(tensor)
    out = enhanced.clamp(0.0, 1.0).squeeze(0).permute(1, 2, 0).numpy()
    return Image.fromarray((out * 255.0).round().astype(np.uint8))


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
