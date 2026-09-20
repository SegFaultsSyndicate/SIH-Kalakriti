# CLAUDE.md — services/ml-svc

Context for a fresh agent picking up this service. Read this before touching
anything -- the previous version of this file described a different,
superseded prototype (`ml_svc/features/*.py`, a `craft_ontology.json` v2.0.0
schema, batch RPCs, an `exec_log.py`) that does not exist anywhere in this
tree. What follows describes the actual code, as of the per-component
modularization pass (see git log for "Modularize ml-svc's model layer").

## What this service is

Kalakriti's ML gRPC inference service (`inference.v1.InferenceService`, see
`proto/inference/v1/inference.proto` at the repo root), called synchronously
by the bff/core-svc. Nine RPCs: `AssessImageQuality`, `EnhanceImage`,
`ExtractAttributes`, `GenerateDescription`, `VerifyTechnique`,
`DetectHandloom`, `Embed`, `Rerank`, `Transcribe`.

## Architecture: three layers below the wire

```text
server.py            protobuf <-> plain Python; builds features from a Registry
    |
features/<rpc>.py    business logic per RPC; orchestrates 1+ model components;
                      no protobuf, no backend-specific imports; the unit meant
                      to be independently unit-testable with hand-written fakes
    |
models/<component>/  one directory per swappable backend; owns its own
                      Protocol, Config, and real.py/mock.py; the unit meant to
                      be hot-pluggable (swap a repo id, an API call, a
                      checkpoint path) without touching anything above it
    |
registry.py + model_loading.py + batching.py   generic loading/scheduling
                      infrastructure, shared by every component
```

**Why two layers below `server.py`**: some models back more than one feature
(the VLM backs `extract_attributes`, `verify_technique`, and
`generate_description`'s polish pass), and some features are pipelines over
several components plus pure logic (`enhance` chains white-balance -> the
lighting component -> the background component -> upscale). Collapsing
models and features into one unit would either duplicate the VLM's loading
three times or recreate a monolith.

**`server.py`'s `InferenceServicer`** is constructed with no components at
all (`__init__(cfg)`) -- grpc.aio requires the servicer registered before
`server.start()`, but loading is slow and must not block the event loop. The
health check stays `NOT_SERVING` while `registry.load_all(cfg)` runs (off the
loop, in `serve()`), then `servicer.attach(registry)` builds every
`features/*.py` object from the loaded components, then the health check
flips to `SERVING`. Every RPC method below that point calls into exactly one
feature object; `server.py` never imports a model component directly.

## The 9 model components (`app/models/<component>/`)

Each directory has the same shape:

```text
app/models/image_background/
    __init__.py   # Protocol, Config dataclass with from_env(), build(cfg)
    real.py       # the actual backend; heavy imports live inside methods,
                   # never at module scope, so mock mode never needs them
    mock.py       # deterministic fake, zero heavy deps
```

`build(cfg: Config)` (the *global* `Config` from `app/config.py`, passed to
every component) picks `mock.py` or `real.py` off `cfg.mock_mode`, calling
its own component-level `Config.from_env()` for whatever env vars it owns.
This is the one idiom every component follows -- copy it exactly when adding
a tenth.

| Component | Backs | Real implementation | Canonical env var(s) |
|---|---|---|---|
| `storage` | every RPC that touches media | MinIO | `ML_SVC_OBJECT_STORAGE_ENDPOINT`/`_ACCESS_KEY`/`_SECRET_KEY`; bucket is `ML_SVC_OBJECT_STORAGE_BUCKET`, read into the *global* `Config.media_bucket` |
| `image_quality` | `AssessImageQuality` | Classical CV, no model weights: PIL decode + Laplacian-variance blur score | `ML_SVC_IMAGE_QUALITY_MIN_WIDTH_PX`/`_MIN_HEIGHT_PX`/`_MIN_BLUR_VARIANCE`/`_MAX_BLANK_STDDEV` |
| `image_background` | `EnhanceImage` | rembg / BiRefNet | `ML_SVC_IMAGE_BACKGROUND_MODEL` |
| `image_lighting` | `EnhanceImage` | Zero-DCE++ (`net.py`, the old top-level `zero_dce.py`) | `ML_SVC_IMAGE_LIGHTING_MODEL` (a checkpoint path; empty = optional, degrades to unavailable) |
| `vlm` | `ExtractAttributes`, `VerifyTechnique`, `GenerateDescription`'s polish | Qwen2.5-VL, one checkpoint backs all three; two interchangeable backends (`real.py`, in-process transformers+bitsandbytes; `llamacpp.py`, an HTTP client to a `llama-server` sidecar over GGUF -- see `ML_SETUP.md` §5 for why both exist) | `ML_SVC_IMAGE_ATTRIBUTE_EXTRACTION_MODEL`, `ML_SVC_TECHNIQUE_VERIFICATION_MODEL`, `ML_SVC_IMAGE_DESCRIPTION_MODEL` (all default to the same repo id; see `vlm/real.py`'s docstring for what happens if they diverge), `ML_SVC_VIDEO_FRAMES`, `ML_SVC_VLM_BACKEND` (`transformers`\|`llamacpp`), `ML_SVC_VLM_LLAMACPP_URL` |
| `embedding` | `Embed` | SentenceTransformer (e5), ONNX Runtime by default | `ML_SVC_TEXT_EMBEDDING_MODEL`, `ML_SVC_TEXT_EMBEDDING_BACKEND` (`onnx`\|`torch`) |
| `reranking` | `Rerank` | CrossEncoder (bge), ONNX Runtime by default | `ML_SVC_TEXT_RERANKING_MODEL`, `ML_SVC_TEXT_RERANKING_BACKEND` (`onnx`\|`torch`) |
| `handloom_texture` | `DetectHandloom` | FFT peak-ratio + an optional, never-yet-trained texture CNN (loader now wired, no checkpoint exists yet) | `ML_SVC_HANDLOOM_DETECTION_MODEL` (checkpoint path; empty = optional, degrades to FFT-only, same pattern as `image_lighting`) |
| `voice_transcription` | `Transcribe` | Bhashini HTTP API -- proof "a model" can be an external API call, not just a local checkpoint | `ML_SVC_VOICE_TRANSCRIPTION_URL`/`_API_KEY` |

**`embedding`/`reranking` run through ONNX Runtime, not eager PyTorch, by
default** (`backend="onnx"` on both `EmbeddingConfig`/`RerankingConfig`,
passed straight into `SentenceTransformer(...)`/`CrossEncoder(...)`). Unlike
the VLM, these are static-graph encoder models with no autoregressive
decode, so ONNX export is mature and low-risk here -- the same category of
win `image_background` already gets from rembg shipping ONNX weights, just
applied via sentence-transformers' native `backend` kwarg instead of a
vendor library. Set `ML_SVC_TEXT_EMBEDDING_BACKEND`/
`ML_SVC_TEXT_RERANKING_BACKEND` to `torch` to fall back to eager mode. The
VLM stays on plain `transformers` -- see the "Adding or swapping a model"
section below for why ONNX doesn't fit it.

**A repo with no pre-published `onnx/model.onnx` on the Hub (e.g.
`BAAI/bge-reranker-v2-m3`; `intfloat/multilingual-e5-base` happens to have
one) does NOT get its on-the-fly export persisted by sentence-transformers
itself** -- confirmed empirically, it logs "heavily recommended to save...
push_to_hub" and holds the export in memory only, so a naive `backend="onnx"`
call would silently re-export from scratch on every process restart, a real
and avoidable cost, not just a one-time one. `app/model_loading.py`'s
`load_or_export_onnx`/`onnx_export_dir`/`save_onnx_export_if_needed` fix
this: both `embedding/real.py` and `reranking/real.py` check
`$HF_HOME/onnx-exports/<repo-id-with-double-dash>/` first and load straight
from there if it exists (no download, no export), otherwise load+export
normally and save into it for next time. Harmless no-op for a repo that
already had a Hub-published onnx file (just a redundant local copy) --
callers don't need to know which case applied. Verified both paths manually
(first load exports+saves; second load skips straight to the saved copy)
since there's no automated test for real-mode loading (see the Tests
section below).

**`onnxruntime` is pinned to `1.22.0` in `pyproject.toml`, not left to
`sentence-transformers[onnx]` extra's own resolution** -- confirmed
empirically, anything newer (`1.29.0` as of this pin) crashes every
real-mode embedding/reranking load with `AttributeError: module 'torch' has
no attribute 'int4'`. `onnxruntime`'s `transformers/io_binding_helper.py`
unconditionally references `torch.int4` in a type-mapping dict regardless of
whether the model actually uses int4, and `torch.int4` doesn't exist before
torch 2.5 -- this project pins `torch==2.4.0` for Qwen2.5-VL/bitsandbytes
compatibility (see the `vlm` row above), so bumping torch to fix this would
risk the VLM's carefully-pinned quantization stack instead. Bisected
1.17.3-1.29.0 by hand; 1.17.3 through 1.22.0 lack the offending entries. Same
pin applies to `onnxruntime-gpu` in the `onnx-gpu` extra.

**`handloom_texture`'s checkpoint loader is now wired** (`net.py` +
`real.py`'s `_get_net()`, copied from `image_lighting/real.py`'s lazy-load
pattern exactly: unset or bad checkpoint logs and degrades, never raises).
Still no checkpoint to actually point `ML_SVC_HANDLOOM_DETECTION_MODEL` at
-- checked before writing `net.py`: no public handloom-vs-powerloom
checkpoint or dataset exists anywhere. Two papers doing exactly this binary
classification (gamucha towels, Sci Rep 2024; Mekhela Sador saris, NMITCON
2024) both built private, unreleased datasets (100-200 physical fabric
samples, phone-camera macro shots at 5-10cm, augmented to 17k+ images) and
both trained a small/modified net from scratch rather than reusing a
published one. The first paper is the more useful data point for anyone
tempted to reach for a pretrained ImageNet backbone here instead of
collecting task data: it benchmarks VGG16/19, ResNet50, InceptionV3, and
DenseNet201 against a small custom net on this exact task, and the
pretrained backbones scored WORSE (50-92% val accuracy) than the ~11M-param
custom net (94-98%) -- unsurprising, since weave regularity is a local,
frequency-domain property (what the FFT peak-ratio already measures
analytically), not the object-shape semantics ImageNet pretraining teaches.
`net.py`'s `TextureNet` is therefore sized for training from scratch on a
small dataset (a few thousand parameters, depthwise-separable convs,
`AdaptiveAvgPool2d` so it accepts any crop size), not for loading a
pretrained backbone. Dataset bootstrap plan, once real listing photos exist:
weak-label a pile of unlabelled weave close-ups with the existing FFT
peak-ratio (confident tails = provisional label, human only adjudicates the
ambiguous middle), which is a much smaller labelling job than starting cold
-- see `real.py`'s module docstring.

**Also fixed while wiring the above**: `score()` passed its FFT input
(`pixels`, already mean-subtracted for the FFT) into `_texture_cnn_score`,
which then divided by 255 expecting ordinary `[0, 255]` greyscale -- inert
while `_texture_net` was always `None` unconditionally, but would have fed
the net garbage the moment a checkpoint was configured. Fixed by keeping the
pre-mean-subtraction array (`raw_pixels`) around and passing that to the CNN
instead.

**Every one of these env var names changed in the modularization pass** (from
vendor/backend-shaped names like `ML_SVC_VLM_MODEL`, `S3_*`, `BHASHINI_*` to
the feature-oriented names above). Nothing else in this repo referenced the
old names (checked `docker-compose.yml`, `.env.example`, `docs/`) -- ml-svc's
compose block only sets `ML_SVC_GRPC_PORT`/`ML_SVC_METRICS_PORT`/
`ML_SVC_MOCK_MODE` and runs in mock mode, so none of the component-specific
vars needed updating there. If you add a real-mode deployment config, use the
table above, not anything from git history before this pass.

## Global config vs. component config (`app/config.py`)

`app/config.py`'s `Config` carries only truly global settings: `grpc_port`,
`metrics_port`, `mock_mode`, `max_batch_size`, `max_wait_ms`, `log_level`,
`model_version`, `craft_allowlist_path`, `media_bucket`. Everything else --
a component's repo id, checkpoint path, or API endpoint -- lives in that
component's own `Config` dataclass, read via that component's own
`from_env()`, called from inside its own `build(cfg)`. `load_craft_allowlist`/
`load_craft_vocab` (reading `scripts/data/crafts.csv`) also stay in
`app/config.py` since they're used by both the `extract_attributes` feature
and the `vlm` mock backend, not owned by any one component.

## The pure-logic layer (`app/features/templates.py`)

`_template_sentence`/`_template_title`/`_template_highlights`/
`_template_keywords`/`_is_grounded`/`_vocab_hint`/`_closed_vocab`/
`_same_technique` are plain functions, no model, no I/O, shared by
`extract_attributes.py`, `verify_technique.py`, and `generate_description.py`.
They run identically regardless of which backend (real or mock) produced the
raw proposal they're checking -- **this is the load-bearing design decision**
in this refactor: a real component's hallucination and a mock component's
fabrication both flow through the exact same guard (allowlist/vocabulary
enforcement, the description grounding check), so there is no separate
"mock-mode logic" to keep in sync with "real-mode logic." Concretely:

- `vlm.propose_attributes(...)` returns a *raw*, unenforced guess (craft id,
  material, technique, colours, motifs, confidence). `extract_attributes.py`
  applies the craft-allowlist fallback and per-craft closed-vocabulary check
  itself, on whatever the component handed back. The mock backend already
  only ever proposes an allowlisted craft and (when given a craft's
  vocabulary) an on-vocabulary material/technique, so this enforcement pass
  is normally a no-op for mock and a real guard for real -- by design, not
  by accident.
- `vlm.observe_technique(...)` returns a raw `matches` verdict from the model
  itself; `verify_technique.py` ANDs it with `_same_technique(observed,
  claimed)`, a pure word-overlap check, so a model claiming a match while
  describing something unrelated still gets caught.
- `vlm.polish(...)` returns a raw rewrite; `generate_description.py` checks
  `_is_grounded` and falls back to the deterministic template sentence if
  the rewrite dropped or swapped a cited value. Mock's `polish()` is a
  no-op passthrough of the template -- grounded by construction, since
  nothing about it changed.

## Images/media cross component boundaries as raw bytes, not PIL objects

Every component that touches pixels (`image_background`, `image_lighting`,
`vlm`, `handloom_texture`) takes and returns plain `bytes`, decoding to PIL
internally only inside its own `real.py`, only when the bytes are non-empty.
`app.models.storage`'s mock backend always returns `b""` (mock mode has no
real content anywhere), and every pure PIL op in `features/enhance.py`
(white balance, upscale) guards on `if not data: return data`. This is what
keeps mock mode free of PIL/torch/cv2 at both the component layer and the
one feature (`enhance.py`) that does real, un-swappable PIL work directly --
without needing a single `if mock_mode` check anywhere in `features/`.
Verify this invariant still holds with `python -c "from app import server"`
using only the base (non-`real`-extra) install.

`app/model_loading.py` (device/dtype resolution, HF-cache-aware
`from_pretrained` kwargs) is unchanged by this refactor and is reused by
`embedding/real.py`, `reranking/real.py`, and `vlm/real.py`. `app/batching.py`
(`MicroBatcher`) is also unchanged -- `server.py`'s `InferenceServicer` still
batches `Embed`/`ExtractAttributes` through it, now calling
`self._embedder.embed(...)`/`self._extractor.extract(...)` (feature objects)
instead of a monolithic registry.

## Tests

- `tests/models/test_<component>_mock.py` -- one per component, cheap, no
  torch, mirrors what `tests/test_mock_models.py` used to cover all at once.
- `tests/models/test_image_lighting_net.py` -- the Zero-DCE++ architecture
  shape/state-dict tests, `importorskip("torch")`.
- `tests/models/test_image_lighting_real.py` -- the one real-mode behaviour
  that needs no torch at all (unconfigured checkpoint -> unavailable).
- `tests/models/test_handloom_texture_net.py` / `test_handloom_texture_real.py`
  -- same two-test split as `image_lighting`'s pair, same reasons.
- `tests/models/test_image_quality_real.py` -- exercises every real check
  (corrupt, empty, too-small, blank, blurry, sharp-passes) against actual
  generated images, `importorskip`d on PIL/numpy/cv2 same as the pairs above.
- `tests/models/test_vlm_llamacpp.py` -- request-shape and response-parsing
  tests against a monkeypatched `httpx.post`, `importorskip`d on httpx; no
  real `llama-server` needed.
- `tests/features/test_<feature>.py` -- one per feature, using hand-written
  fake Protocol implementations, not any component's real mock. Mirrors what
  `tests/test_real_helpers.py` used to cover as one flat file of pure-function
  tests plus some real.py-specific ones.
- `tests/features/test_templates.py` -- the shared pure-logic layer.
- `tests/test_config.py` -- `load_craft_vocab`'s CSV-parsing edge cases.
- `tests/test_server_mock.py` -- unchanged in spirit: every RPC over a real
  gRPC connection in mock mode, now built from `Registry` via `attach()`.
  This is the strongest signal that the internal reshuffle didn't change
  external behaviour.
- `tests/test_batching.py` -- untouched, `MicroBatcher` was already
  model-agnostic.

Run with `.venv/bin/python -m pytest` after `uv sync --extra dev` and
`./scripts/gen_proto.sh` (see that script's own docstring for why `pb/` has
to be generated before the suite can import `app.pb`/run
`test_server_mock.py`; two of its tests `importorskip` without it).

## Adding or swapping a model: what should change

- **Swapping a backend for an existing capability** (e.g. `image_description`
  pointing at Gemini instead of the local VLM): add `app/models/vlm/gemini.py`
  implementing the `VisionLanguageModel` Protocol, add a `backend` selector to
  `VLMConfig`/`build(cfg)`. Zero changes to `features/generate_description.py`
  or `server.py` -- this is the whole point of the Protocol boundary.
- **Why the VLM doesn't get the same ONNX treatment as `embedding`/
  `reranking`**: those two are static-graph encoders; the VLM is
  autoregressive generation over a vision tower plus a decoder with a
  dynamic KV cache, a shape ONNX Runtime and Optimum's export tooling don't
  handle well for Qwen2.5-VL as of `transformers==4.49.0`.
- **The VLM's dedicated-serving-runtime path is `vlm/llamacpp.py`, not
  vLLM.** vLLM was evaluated first and rejected for this project's actual
  hardware: it pre-allocates a fixed fraction of *total* GPU memory before a
  single request (PagedAttention/continuous batching's whole point is high
  concurrency, which this service doesn't need locally), and even a 4-bit
  AWQ checkpoint of this exact model doesn't fit a 4GB card -- AWQ quantizes
  only the LLM body, leaving the ~0.67B vision tower at fp16, so weights
  alone (~3.4GB) exceed usable VRAM before any KV cache exists. llama.cpp's
  GGUF format is the only one of the runtimes evaluated (vLLM, SGLang,
  llama.cpp) that can quantize the vision tower itself
  (`mmproj-...-Q8_0.gguf`) and evict it to CPU entirely
  (`--no-mmproj-offload`) -- see `docs/ML_SETUP.md` §5 for the actual setup
  and the small-card tuning flags. `real.py` (transformers+bitsandbytes)
  stays the default and the better fit for genuinely tiny cards; `llamacpp.py`
  is what to reach for once real GPU headroom (8GB+) is available, or to get
  llama.cpp's `response_format: json_schema` structured-output guarantee
  instead of `real.py`'s regex-based `_parse_json`.
- **Adding a ninth component**: new `app/models/<name>/` package following the
  shape above, one `load(...)` call added to `app/registry.py`'s `load_all`
  and one field added to `Registry`, a new or extended `features/<rpc>.py`.
- **Never**: reach for a plugin-discovery/auto-registration system. The
  explicit, named list in `registry.py` is deliberate -- it is the one place
  that answers "what backends does this service load," by inspection, not by
  scanning a directory at runtime.

## Standing constraints (carried over, still true)

- **Never run installs or downloads that pull heavy files/models.** State the
  exact command (`uv add <pkg>` then `uv sync`, or the HF repo id / checkpoint
  URL) and stop -- let the user run it themselves.
- Mock mode must stay free of torch/transformers/PIL/cv2/rembg/minio/httpx at
  both import time and call time -- see the "raw bytes, not PIL objects"
  section above for how that invariant is actually maintained through this
  refactor's component boundaries.
- Every real component's blocking call is either wrapped by its calling
  feature in `asyncio.to_thread` (see any `features/*.py`) or, for
  `voice_transcription`, is itself genuine async I/O (httpx streaming) --
  grpc.aio runs all RPCs on one event loop, so a direct synchronous call
  blocks every other in-flight request, including health checks. This is
  enforced by convention, not a lint rule -- check by eye when adding a new
  feature or component.
