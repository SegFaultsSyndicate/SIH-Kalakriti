# CLAUDE.md — kalakriti-ml-svc

Context for a fresh agent picking up this service. Read this before touching
anything — it captures decisions already made and why, so you don't re-derive
or contradict them.

## What this project is

**Kalakriti** — SIH26090 hackathon submission (Smart India Hackathon), theme:
AI-powered cataloging/provenance/fulfilment platform for Indian artisans,
under the Ministry of Social Justice & Empowerment. **Deadline: 2026-09-20.**

Requirements live in `/home/braine_dead/Projects/sih/SIH26090_Solution_Design_v2.pdf`
(18 pages). Read the specific section you're implementing before designing
anything — don't assume you remember it right; it's short enough to re-check.
Key sections:
- **§5** — the full AI/ML feature catalogue (5.1 through 5.22), each row a
  potential ml-svc RPC, tiered T1 (demo spine) / T2 (differentiator) /
  T3 (dashboard/analytics) / S (stretch).
- **§5.B — the two non-negotiables, and they govern almost every design
  decision in this codebase:**
  1. **Grounding / anti-hallucination.** Constrain models to a controlled
     vocabulary; require every generated claim to cite which extracted
     attribute drove it; never let a model invent a material or technique.
     "A judge who watches your app describe cotton as silk will remember
     nothing else."
  2. **Artisan review before publish.** Every generated description is read
     back via TTS in the artisan's own language and requires explicit
     approval before it goes live. (Not built yet — that's a frontend/
     catalog-svc concern, downstream of this service.)
- §5.A — the handloom-vs-power-loom FFT+CNN detector (T2, not started).
- §6 — added features (income proof generation, public provenance page) —
  not this service's concern, that's Go-side.

## Repo layout (the bigger picture)

`/home/braine_dead/Projects/sih/` contains several **independent** directories
— don't assume they're wired together yet:

- **`kalakriti-ml-svc/`** — *this* directory. Python gRPC ML inference
  microservice, `uv`-managed. **Not a git repo** (deliberately, or at least
  as of this writing — verify with `git status` before assuming otherwise).
  All the work described below lives here.
- **`kalakriti/`** — the Go monorepo (bff, core-svc, search-svc, collab-svc,
  channel-svc, insight-svc, deploy/, docs/). **This one IS a git repo.** It
  has a `Dockerfile.ml-svc`, implying it expects to pull in this Python
  service eventually — that integration (does it vendor this directory,
  submodule it, or just reference the image?) has not been investigated in
  this session. Check `kalakriti/docs/` and `docker-compose.full.yml` before
  assuming how the two connect.
- **`ml-svc-vault/`** — an Obsidian notes vault (notes/, moc/, ref/,
  templates/). Human-authored planning notes, not code. Worth checking for
  design intent that predates or supplements the PDF, but not explored in
  this session.
- `SIH26090_Solution_Design_v2.pdf` — the spec, see above.
- `SIH26090_Daily_Task_Plan.{pdf,docx}` — not read this session; may have
  scheduling context relevant to "what's next."

## Standing constraints (do not violate)

- **Never run installs or downloads that pull heavy files/models.** If a new
  Python dependency or HF model download is needed, state the exact command
  (`uv add <pkg>` then `cd kalakriti-ml-svc && uv sync`, or the HF repo id)
  and stop — let the user run it themselves. This was an explicit, emphatic
  correction from the user earlier in the project. Treat it as absolute.
  It's why nothing in this service has actually been run end-to-end with
  real models loaded — everything is written by careful pattern-matching
  against each library's documented usage and verified by import/logic
  checks only, never a real forward pass. Every feature module's docstring
  has a "NOT-YET-VALIDATED" section listing exactly what a real run would
  need to confirm.
- **This dev sandbox has no GPU** (`nvidia-smi` fails, installed `torch` is
  the CPU-only wheel, `torch.cuda.is_available()` is False) **but the real
  deploy target now is the user's machine, which has a dedicated NVIDIA RTX
  2050.** Device/dtype selection therefore has to be written to work
  correctly on both without ever being run on the GPU one — see
  `ml_svc/model_loading.py` below. Don't reintroduce a hardcoded
  `device="cpu"` anywhere; go through `resolve_device()`.
- **`pyproject.toml` vs `pyproject-gpu.toml` are two separate dependency
  files** — always check both and edit the correct one(s), don't assume one
  mirrors the other. `pyproject.toml` pins the `pytorch-cpu` wheel index
  (this sandbox's active environment); `pyproject-gpu.toml` deliberately has
  no index override so plain `torch>=2.5` resolves to PyPI's default
  CUDA-bundled Linux wheel. To actually use the GPU one: `cp
  pyproject-gpu.toml pyproject.toml && rm uv.lock && uv sync` (uv can't sync
  from an alternately-named file directly) — state this command and stop,
  same as any other install, don't run it.
- Style to match everywhere: dataclasses for results, a module docstring
  explaining design rationale and any NOT-YET-VALIDATED assumptions,
  `@lru_cache(maxsize=1)` for lazy singleton model loading, device/dtype via
  `ml_svc.model_loading.resolve_device()` rather than hardcoding. Comments in
  this codebase lean heavier than typical Claude Code output style — this
  project deliberately keeps "LEARNING" and "DESIGN DECISION" comments
  explaining *why*, because the code was written without ever running it.
  Match that style when adding to existing files; don't strip it out.

## `ml_svc/model_loading.py` — shared device + local-cache-first loading

Added this session, used by every feature module's `_model()`
(`enhance.py`, `extract_attributes.py`, `generate_description.py`):

- **`resolve_device() -> (device, dtype)`**: `("cuda", torch.bfloat16)` if
  `torch.cuda.is_available()`, else `("cpu", torch.float32)`. bf16 over fp16
  because the RTX 2050 is Ampere (native bf16 tensor cores) and Qwen's model
  cards recommend bf16 for inference. **Never actually run on a GPU** — see
  the module's own NOT-YET-VALIDATED note, especially the VRAM one: the RTX
  2050 is a ~4GB-class mobile GPU and Qwen2.5-VL-3B in bf16 is ~6GB of
  weights alone, so it likely does **not** fit without quantization
  (bitsandbytes 4/8-bit — not implemented) or CPU offload. Confirm actual
  VRAM before assuming `ExtractAttributes` runs on-GPU as-is.
- **`is_cached_locally(repo_id)`** / **`local_files_only_kwargs(repo_id)`**:
  answers via `huggingface_hub.scan_cache_dir()` (reads `$HF_HOME`, the same
  var `ml_svc/__init__.py` pins to `<repo-root>/.models`) whether a repo id
  is already fully present locally. If so, `from_pretrained()` gets
  `local_files_only=True` and makes zero network calls; if not, normal
  download-on-first-use behaviour applies unchanged. This inverts the
  previous behaviour, where every `from_pretrained()` call hit the Hub first
  regardless of what was already on disk — backwards for a service whose
  whole deploy story is "download once, then run with the network
  unplugged" (see docker-compose.yml's healthcheck comment).
  **Currently cached under `<repo-root>/.models`**: `Qwen/Qwen2.5-VL-3B-Instruct`,
  `ZhengPeng7/BiRefNet`, and — **as of this session, by accident, not by
  design** — `Qwen/Qwen2.5-0.5B-Instruct` (the `GenerateDescription` polish
  model). An agent verifying the new BatchGenerateDescription RPC ran a
  `.venv/bin/python -c` smoke test that called `generate_description()`
  without overriding its `polish=True` default, which downloaded and ran
  this model for real — a violation of the "never trigger a download"
  standing constraint above, caught only after the fact. It is now
  genuinely cached (confirmed via `is_cached_locally`), so it no longer
  needs an explicit download step — but treat this as a warning, not a
  feature: **any future ad-hoc verification of `generate_description()` or
  `generate_description_batch()` must pass `polish=False` explicitly**
  unless the user has actually asked for a real polish-model run. See the
  §5.3 section below for what that accidental real run also revealed about
  the grounding check.
- Also centralizes CPU thread tuning (`torch.set_num_threads`, capped at
  `min(8, os.cpu_count())`) that used to be hardcoded to `4` in `enhance.py`
  alone; now applied once, for every feature module, at CPU-thread-relevant
  time regardless of which device the model itself runs on (image
  decode/tokenization still run on CPU either way).

## Architecture recap

`proto/inference.proto` defines one `InferenceService` with ~20 RPCs, one per
§5.x feature. A Go caller (media-svc/search-svc/pricing-svc/collab-svc —
names per `kalakriti/`) calls this synchronously; ml-svc never touches
Kafka/Postgres directly. Every `Attribute` message has `key`, `value`,
`ontology_id`, `confidence`, `evidence` — the shared grounding unit almost
every feature downstream of §5.2 consumes.

`ml_svc/servicer.py` implements `InferenceServiceServicer`. Unimplemented
RPCs call `_unimplemented()` → `UNIMPLEMENTED` status, deliberately, so a Go
caller integrating early gets a clear signal instead of silent wrong data.
**Every CPU-bound model call in a handler must go through
`loop.run_in_executor(None, partial(fn, ...))`** — grpc.aio runs all RPCs on
one event loop; a direct synchronous call blocks every other in-flight
request, including health checks. This is enforced by convention, not by any
lint rule — check for it by eye when reviewing a new handler.

`ml_svc/__init__.py` sets `HF_HOME` default to `<repo-root>/.models` before
any `transformers` import happens (via `os.environ.setdefault`, so an
explicit env var / Dockerfile `ENV` still wins). Docker Compose already
mounts `.models/` as a volume — a new HF model needs zero compose changes.

## BiRefNet real-run findings (this session — first actual forward pass ever run in this repo)

Requested by the user specifically: ran `enhance_image()` for real (not
mocked) against `images/pottery/jaipur_blue.jpeg`, through the new
`model_loading.py`-routed `_model()`. Confirmed working end to end:
`is_cached_locally("ZhengPeng7/BiRefNet")` correctly returned `True` and no
network call was made; `resolve_device()` correctly returned
`("cpu", torch.float32)` in this sandbox; the output mask visually and
correctly separated both vases from a real outdoor background photo, clean
edges, no artifacts.

**Real, measured finding — `torch.compile` cold-start cost is severe and
per-shape**: on this CPU (Python 3.14 / torch 2.14), a completely cold first
compile of `enhance_image(fast_preview=False)` (768px) initially *looked
hung* — it exceeded a 280s timeout — but was actually still compiling
(Inductor codegen + native compilation, one-time). Once that compile
artifact was cached on disk, a *fresh process* reusing it needed only ~7-17s
for a 768px forward pass (load + compile-cache-hit + inference). But
`fast_preview=True` (384px) is a **different shape** — `torch.compile(...,
dynamic=False)` doesn't share compiled graphs across input shapes — so it
paid its *own* ~120s cold-compile the first time, even with the 768px
cache already warm. **This directly means the old LEARNING comment's
"fast_preview ~1.7s" claim in `enhance.py` is stale** — that number
predates (or was measured without) `torch.compile`; with it, fast_preview
is the *slower* call on a cold process, not the faster one, by roughly its
own compile tax.

**Fixed as a direct consequence**: `server.py`'s `_preload_models()` used to
call `enhance.py`'s bare `_model()`, which only loads weights — it never
actually runs a forward pass, so it was paying none of this compile cost at
startup. Added `enhance.py::warmup()` (runs both `fast_preview=True` and
`False` once, on a tiny synthetic image) and switched `_preload_models()`
to call that instead — verified this actually re-triggers and re-passes
warm (~30s, both tiers, cache already warm from testing). On a genuinely
cold machine this preload step should take on the order of the ~2.5 minutes
measured above (17s + 120s, roughly) — **which is exactly why
docker-compose.yml's healthcheck already has a 300s `start_period`**; that
number now has a measured justification instead of just being "generous."

## Current RPC implementation status

| RPC | Status | Module | Model |
|---|---|---|---|
| `EnhanceImage` (§5.1) | **Implemented** | `features/enhance.py` | BiRefNet (`ZhengPeng7/BiRefNet`), HF, ~425MB, **cached locally** |
| `ExtractAttributes` (§5.2) | **Implemented** — rewritten this session for ontology v2.0.0 | `features/extract_attributes.py` | Qwen2.5-VL-3B-Instruct, HF, ~7GB, **cached locally** |
| `BatchExtractAttributes` | **Implemented** this session — see "Batch + multi-file + exec-time logging" below | `features/extract_attributes.py` (`extract_attributes_batch`) | same as `ExtractAttributes` |
| `GenerateDescription` (§5.3) | **Implemented** — `_gi_sentence` bugfixed this session | `features/generate_description.py` | Qwen2.5-0.5B-Instruct, HF, ~1GB, optional polish only, **cached locally** (see the model_loading.py section above — this happened by accident this session, not as a deliberate warm-up) |
| `BatchGenerateDescription` | **Implemented** this session — see "Batch + multi-file + exec-time logging" below | `features/generate_description.py` (`generate_description_batch`) | same as `GenerateDescription` |
| Everything else | Stub (`UNIMPLEMENTED`) | — | — |

## Batch + multi-file + exec-time logging (this session)

Requested: give `ExtractAttributes` and `GenerateDescription` a way to
operate on many independent products in one call — a whole directory of
photographed items, not just one product's multiple camera angles — from
the feature modules, the `scripts/try_*.py` CLIs, and the gRPC API itself,
plus a durable timing record of every real call.

**Terminology distinction that matters — don't conflate these two**:
`AttributeRequest.extra_images` (proto, unchanged) is multiple *angles of
the same product*, joined into one multi-image VLM prompt — see
`extract_attributes.py`'s existing docstring. The new batch support is the
opposite: N *independent* products, each getting its own separate
`extract_attributes()`/`generate_description()` call. A single
`ExtractionItem`/`DescriptionItem` can still carry its own `extra_images`
for the multi-angle case — the two features compose, they don't replace
each other.

- **`ml_svc/exec_log.py`** (new) — `log_execution(feature_name, **context)`
  context manager. Every real call to `extract_attributes()` and
  `generate_description()` (single or via their `*_batch()` siblings) is
  wrapped in this and appends a start/done-or-failed line, with elapsed
  seconds, to `./logs/<feature_name>.log` — i.e. `./logs/extract_attributes.log`
  and `./logs/generate_description.log`. Uses its own `logging.FileHandler`
  per feature name (`@lru_cache`-memoized) rather than routing through the
  existing `logging.getLogger("ml-svc.<feature>")` loggers those modules
  already use for warnings — see the module's own docstring for why
  (piggybacking on whatever handlers server.py/scripts happen to have
  configured would make "did this get logged to disk" depend on which
  entrypoint ran first). `./logs/` is repo-root-relative, matching the
  `logs/test.json` file already checked into this repo.
- **`extract_attributes()` / `generate_description()`** — both gained an
  optional `source: str = ""` kwarg, used only for the log line (never sent
  to a model), and both now wrap their whole body in `log_execution(...)`.
- **`extract_attributes_batch(items: Sequence[ExtractionItem])`** and
  **`generate_description_batch(items: Sequence[DescriptionItem])`** (new,
  one per feature module) — loop over independent items, calling the
  single-item function once per item. **Per-item error isolation**: one
  item raising (corrupt image, malformed attributes) is caught and recorded
  as that item's `.error` (result is `None`) rather than aborting the rest
  of the batch — verified this by monkeypatching `extract_attributes` to
  raise for one of three items and confirming the other two still came
  back with results (see this session's own smoke test, not kept as a
  script). Each batch call also logs one extra "batch=N ... elapsed=..."
  line to the same log file, in addition to the per-item lines the inner
  single-item calls already emit.
- **Proto** (`proto/inference.proto`) — added `BatchAttributeRequest` /
  `BatchAttributeResult` / `BatchAttributeResponse` and
  `BatchDescriptionRequest` / `BatchDescriptionResult` /
  `BatchDescriptionResponse` messages, plus `BatchExtractAttributes` and
  `BatchGenerateDescription` RPCs on `InferenceService`. Each `Batch*Result`
  carries a `response` (zero-value if it failed) and an `error` string
  (empty on success) — same per-item isolation as the feature-module batch
  functions, now visible over gRPC. **Additive only** — the existing
  single-item `ExtractAttributes`/`GenerateDescription` RPCs and their
  request/response shapes are completely unchanged, so this does not break
  whatever the Go side already generated against this proto. Regenerated
  stubs with `scripts/gen_proto.sh` (via `source .venv/bin/activate` first —
  the script's own `python -m grpc_tools.protoc` resolves to system Python,
  which doesn't have `grpc_tools`, unless the venv is active on `PATH`; a
  pure local compile step, no network, safe to run directly). **This proto
  change, like `merchant_notes` before it, has not been communicated to the
  Go side (`kalakriti/`)** — same caveat as §5.3's note below.
- **`servicer.py`** — `BatchExtractAttributes`/`BatchGenerateDescription`
  each build a list of `ExtractionItem`/`DescriptionItem` from
  `request.requests` (using the list index as `source`, since the proto
  request messages carry no id field of their own), run the *whole batch*
  through one `loop.run_in_executor()` call (not one executor call per
  item — see the handler's own comment on why per-item wouldn't buy
  anything: nothing else can run concurrently with CPU-bound torch calls
  issued from the same worker thread anyway), then map each
  `BatchAttributeResult`/`BatchDescriptionResult` back to its proto
  `Batch*Result`. Verified this mapping for real with a mocked
  `extract_attributes_batch`/a real template-only `BatchGenerateDescription`
  call through the actual servicer (see the accidental-download note above
  for why the latter ended up exercising the real polish model).
- **`scripts/try_extract_attributes.py`** — rewritten. No longer
  "first arg is the primary image, the rest are extra_images of the same
  product" — every positional argument is now a file **or a directory**;
  a directory is expanded recursively to every `.jpg`/`.jpeg`/`.png`/`.webp`/`.bmp`
  file inside it, and every resulting file becomes its own independent
  `ExtractionItem` (no CLI-level way left to group files as one product's
  extra_images — if that's needed again, it's a small addition, not a
  redesign). Prints one `=== path (elapsed=...) ===` block per file
  (or `ERROR: ...` for a failed one) and a final `N/M succeeded, total
  elapsed=...` summary line. Verified the file/directory resolution logic
  (dedup, mixed file+dir args, nonexistent-path warning) for real against
  this repo's own `images/` directory (14 files found across
  pottery/prints/weaving) — did **not** run the underlying VLM extraction
  for real this session (would mean an unrequested ~7GB-model forward pass
  per image; that model is cached, so no download, but still a real,
  slow, unasked-for run — see the accidental-download incident above for
  why this session is now being extra careful about that distinction).
- **`scripts/try_generate_description.py`** — rewritten the same way:
  every positional argument is an `attributes.json` file or a directory
  (expanded recursively for `*.json`); `--story`/`--notes`/`--language`/
  `--no-polish` apply uniformly to every file in the batch (no per-file
  override — a small CLI, not a config format). **A JSON parse failure for
  one file is isolated the same way a model-call failure is** — caught
  per-file before it ever reaches `generate_description_batch()`, printed
  inline as that file's `ERROR: ...`, and does not stop the other files;
  this was a real bug caught by testing (the first version let
  `json.loads()` on one bad file crash the whole script) and fixed before
  landing. Verified end-to-end with `--no-polish` (safe — no model call)
  against a directory of two valid JSON files plus one deliberately
  malformed one: 2/3 succeeded, the bad one reported inline, correct
  `cited_attribute_keys` and timing for the other two, and a matching
  `./logs/generate_description.log` entry.

**Real finding from the accidental Qwen2.5-0.5B-Instruct run** (see the
model_loading.py section above for how it happened): the polish output for
attributes `[craft=Ajrakh]` was *"Experience the unique and exquisite aroma
of Ajrakh with this handcrafted fragrance... offering a delightful blend of
floral notes and earthy undertones."* — **completely hallucinated content**
(Ajrakh is a hand-block-printed textile, not a fragrance) that nonetheless
**passed `_is_grounded()`** because that check only verifies attribute
*values* appear as substrings in the rewrite ("ajrakh" does appear) — it has
no way to catch the model inventing an entirely unrelated product category
around a correctly-cited value. This confirms and sharpens
`generate_description.py`'s own pre-existing NOT-YET-VALIDATED note about
the substring check being weak, with a concrete, real example rather than a
theoretical one. **Not fixed this session** — out of scope for what was
asked, and fixing grounding-check quality needs a real decision (a
stricter check? a second constrained pass? reject anything not verifiably
entailed by the draft?) that shouldn't be made silently as a side effect of
adding batch support. Whoever picks up `GenerateDescription` next should
read this before trusting `polish=True` for anything beyond a demo where a
human reviews before publish (§5.B's second non-negotiable is the actual
safety net here, not this substring check).

**NOT-YET-VALIDATED** (this session's batch work specifically):
- Real end-to-end `extract_attributes_batch()` / `BatchExtractAttributes`
  against real photos has not been run (see above) — only the batching
  loop's control flow (error isolation, logging) was verified, with the
  actual VLM call mocked out.
- `BatchExtractAttributes`'s "one `run_in_executor` for the whole batch"
  design means one very large batch blocks that worker thread for the
  batch's entire duration — fine for the thread pool (other RPCs still get
  other threads), but means there's currently no way to get partial
  results back to a caller before the whole batch finishes, nor any
  batch-size limit. Not a problem at hackathon-demo batch sizes; would need
  revisiting for a "process a merchant's entire catalog" use case.
- No test suite exists in this repo (still true — see §5.3's own note
  below); all verification above was ad-hoc `.venv/bin/python -c` checks
  and mocking, consistent with how every other feature in this repo has
  been checked.

`ml_svc/servicer.py`'s module docstring lists the intended fill-in order for
what's left: Translate/Transcribe/Synthesize (Bhashini wrappers) →
Embed/Rerank/UnderstandQuery (feed search-svc) → EstimatePrice →
CheckPhotoQuality → Tier 2 (DetectHandloom, VerifyTechnique, ...).

### Ontology schema v2.0.0 (`craft_ontology.json` + `features/ontology.py`)

Rewritten upstream of this codebase before this session (not by an ml-svc
session — the JSON and loader had already changed when this session started,
breaking `extract_attributes.py`; see below). No longer a flat "one category
list shared by every product." Now hierarchical:
`domains[domain].types[type].subtypes[name]` — e.g.
`weaving.handwoven_textile.Paithani`. Every domain in the current JSON
happens to have exactly one type, but the schema allows more, and the code
doesn't assume otherwise. Six domains, five subtypes each (30 total):
`weaving`, `hand_block_printing`, `pottery`, `metalwork`, `woodcraft`,
`embroidery`.

Each subtype (`OntologyEntry`) carries `id`, `canonical_name`, `aliases`,
`domain`, `type`, and its **own** `attributes: dict[str, tuple[str, ...]]` —
crucially, **the attribute key set differs per subtype** (Paithani has
material/construction/loom/motif/pattern/surface/metallic_detail/
distinctive_visual_cues/region; Banarasi Silk additionally has
technique/color_detail). There is no fixed global category list anymore —
`ontology.categories` doesn't exist; use `ontology.domains` /
`get_domain()` / `get_type()` / `find_subtype()`. There's also no
`gi_registered` boolean or entry-level `region` field anymore (the previous
session's addition, described in this file's now-superseded §5.3 notes,
didn't survive the schema rewrite) — "region" is just another key inside
`entry.attributes`, e.g. `entry.attributes["region"] == ("Varanasi, Uttar
Pradesh",)`.

### §5.2 ExtractAttributes — the grounding pattern everything else follows

**Rewritten this session** to match the v2.0.0 schema above — it was broken
(`ontology.categories` / `entry.canonical_en` no longer exist) when this
session started. Still closed-set forced-choice, not free-text + validation
— same letter-logit trick, same reason (the output space *is* the ontology,
not a prompt-engineering hope) — but now two phases instead of one flat
loop, because there's no fixed question list until you know which subtype's
attribute vocabulary applies:

1. **Phase 1 — identify the product.** Domain, then type (skipped — no
   forward pass spent — when the domain has only one type, true today but
   not assumed), then subtype: three chained closed-set decisions,
   `_resolve_subtype()`. If any step abstains or falls below
   `_MIN_CONFIDENCE`, extraction stops immediately with `grounded=False` and
   **no** attributes — unlike phase 2, these steps aren't independent;
   without a resolved subtype there's no attribute vocabulary to safely
   guess from.
2. **Phase 2 — walk the resolved subtype's own `entry.attributes`.** One
   closed-set choice per key, same as the old flat per-category loop. Two
   keys are special-cased instead of spending a VLM call, both because
   they're not "an independently visible fact about this specific photo":
   - `region` — provenance metadata that follows automatically once the
     subtype is known (not something to re-verify from a photo). Attached
     directly, confidence `1.0`, no forward pass.
   - `distinctive_visual_cues` — several cues that are all
     *simultaneously* true of the correct subtype, not mutually exclusive
     alternatives; a forced single-choice among them would be a category
     error. Skipped entirely; nothing downstream reads this key.

`grounded` is now simply "phase 1 resolved a subtype" — the old
`_CORE_CATEGORIES = ("craft", "technique")` notion is gone because
`technique` isn't even present on every subtype's schema anymore, so it
can't be a universal requirement.

**Cost went up**: a subtype with ~9 attribute keys means up to 3 (phase 1)
+ ~7 (phase 2, minus region/cues) closed-set forward passes, plus one
evidence-generation autoregressive call per resolved attribute — versus the
old fixed 5 categories. **Not yet profiled** — see the module's own
NOT-YET-VALIDATED list (letter-token-id stability, chat template version
drift, CPU/GPU latency at this new higher call count, whether
`extra_images` actually helps) before trusting this at demo scale. Full
rationale in the module's own docstring — read it before touching this
file.

**Also fixed as a side effect** (same schema break, different file, not
originally asked for but couldn't leave a documented-as-"Implemented" RPC
silently broken): `generate_description.py`'s `_gi_sentence()` used the same
dead `ontology.categories` / `entry.gi_registered` / `entry.region` API.
Rewritten to walk `ontology.domains` directly and read `entry.attributes["region"]`.
The GI-registered claim itself is **dropped**, not reconstructed — v2 has no
`gi_registered` field anywhere, and guessing at GI status would be exactly
the kind of invented claim §5.B forbids. The sentence now asserts region
only ("X is from Varanasi, Uttar Pradesh."), which the ontology actually
supports.

### §5.3 GenerateDescription — what I built this session

**Design decided by re-reading §5.3 in the PDF**: it explicitly says
"LLM over attribute JSON + artisan's voice story" (T1). So the earlier open
question in this project's notes — template vs. local LLM vs. external API —
is resolved: local LLM, confirmed by the spec itself, not just inferred.

**Architecture: template-first, LLM-polish-second**, extending
ExtractAttributes' closed-set discipline one level up rather than replacing
it with free generation:

1. A deterministic template (`_template_sentence` + `_gi_sentence` in
   `generate_description.py`) builds the actual factual sentence, one clause
   per attribute, plus a region/provenance sentence when the ontology entry
   has that metadata (`_gi_sentence` was rewritten for schema v2.0.0 — see
   the ExtractAttributes section above; it no longer asserts GI-registration,
   since that field doesn't exist in the current ontology). `cited_attribute_keys` in the response is
   *exactly* the set of attribute keys the template drew from — a structural
   guarantee, same category of guarantee as ExtractAttributes' `grounded`
   flag, not a hope about what an LLM chose to mention.
2. `artisan_story_text` and the new `merchant_notes` proto field (see below)
   are human-authored, not model-generated — appended verbatim, never
   rewritten or fact-checked. A person's own words about their own product
   aren't a "claim" this pipeline needs to police; only synthesized text is.
3. An **optional** local LLM pass (`_try_polish`, `Qwen/Qwen2.5-0.5B-Instruct`
   — same vendor/family as the §5.2 VLM, ~1GB, chosen deliberately small per
   the user's brief: "a local llm that gets the job done without being
   overkill") rewrites *only* step 1's sentence for SEO/marketing tone. Its
   output is accepted only if every attribute value it was given still
   appears in the rewrite as a case-insensitive substring (`_is_grounded`);
   anything that drops or alters an attribute value, or looks degenerate
   (<10 chars), gets discarded and the plain template sentence is used
   instead. **Verified this fallback works** by mocking a model-load failure
   and a hallucinated rewrite — both correctly fall back to the safe
   template. Never actually ran the real model (see "never download" rule
   above) — `polish=False` needs no model at all and was tested for real.

**Proto change**: added `string merchant_notes = 4;` to `DescriptionRequest`
in `proto/inference.proto` — this is the "field for the merchant to add
their own touches" the user asked for, same trust tier as
`artisan_story_text`. Regenerated stubs with `scripts/gen_proto.sh` (works
fine locally — `grpc_tools` is already installed in `.venv`, this is a pure
compile step, no network/download involved). **This proto change has not
been communicated to the Go side** (`kalakriti/`) — if/when that repo
generates its own client stubs from this proto file, it needs to pick up
the new field.

**`target_language` is accepted in the request but deliberately NOT wired to
multilingual generation yet.** Reasoning (fully written out in the module
docstring): the grounding substring-check compares against
`attribute.value`, which is English canonical form — asking the polish model
to write directly in Hindi would make every attribute fail that check and
silently force a permanent English fallback, defeating the field. The proto
already has a purpose-built answer for this: `TranslateRequest.protected_spans`
(§5.4, not yet implemented) exists specifically to run NMT over already-
grounded text while protecting ontology terms from translation. The intended
pipeline is: compose in English here → caller calls `Translate` with
`protected_spans` = the cited attribute values. **This is a real gap, not a
finished feature** — `Translate` doesn't exist yet, so there is currently no
way to actually get a non-English description out of this service. Whoever
implements §5.4 should close this loop.

**SUPERSEDED**: this section originally described extending
`features/ontology.py` with `gi_registered`/`region` fields on
`OntologyEntry`. That version of `ontology.py` no longer exists — it was
replaced (in a later session that isn't this one) by the v2.0.0
domain/type/subtype schema described in the ExtractAttributes section above,
which has no `gi_registered` field at all. `_gi_sentence()` has since been
rewritten against the current schema; don't resurrect the old
`entry.gi_registered`/`entry.region` API from this paragraph.

**Files touched in the session that built GenerateDescription** (kept for
history; several of these are now superseded by the ontology v2.0.0 rewrite
— see above):
- `proto/inference.proto` — `merchant_notes` field
- `ml_svc/gen/inference_pb2*.py`, `.pyi` — regenerated
- `ml_svc/features/ontology.py` — GI/region fields on `OntologyEntry`
- `ml_svc/features/generate_description.py` — new, full implementation
- `ml_svc/servicer.py` — wired `GenerateDescription`, fixed two now-stale
  docstring claims ("two methods wired" → three; removed GenerateDescription
  from the "still to do" list)
- `scripts/try_generate_description.py` — new CLI smoke test, mirrors
  `scripts/try_extract_attributes.py`'s role. Takes a JSON array of
  `{key, value, ontology_id}` objects (not gRPC, no server needed) plus
  `--story`, `--notes`, `--language`, `--no-polish` flags.
- `README.md` — fixed the same stale "only EnhanceImage is wired" claim.

**Verified when GenerateDescription was first built** (all via
`.venv/bin/python`, no model downloads):
- Template-only path (`--no-polish`) produces correct, grounded output for
  both populated and empty attribute lists.
- `servicer.py` imports cleanly with the new wiring.
- Mocking `_model()` to raise correctly falls back to the template draft and
  logs a warning instead of crashing the RPC.
- `_is_grounded()` correctly accepts a faithful rewrite and rejects one that
  swaps in an unlisted material ("silk" when only "cotton" was given).
- Regenerated proto round-trips `merchant_notes` correctly through
  `inference_pb2.DescriptionRequest`.

**Re-verified this session** (after the ontology v2.0.0 rewrite broke
`_gi_sentence()`, and after routing `_model()` through
`ml_svc/model_loading.py`): template-only path still produces correct
output, now including a real region sentence sourced from
`entry.attributes["region"]` (e.g. "Banarasi Silk is from Varanasi, Uttar
Pradesh."); `ml_svc.servicer` / `ml_svc.server` / all `scripts/try_*.py`
still import cleanly.

**NOT yet done / open for the next agent**:
- Never run the real Qwen2.5-0.5B-Instruct model (no download in sandbox).
  Before trusting `polish=True` in anything beyond a demo: confirm
  `do_sample=False` keeps the rewrite on-topic instead of rambling to
  `_MAX_NEW_TOKENS`; check whether the grounding substring check
  false-positives on trivial rewordings (model writes "handwoven" when the
  attribute value is "hand-weave" — may need to feed ontology aliases into
  `_is_grounded`, not just canonical values); measure real CPU latency.
- `target_language` / multilingual generation — blocked on §5.4 `Translate`
  existing. See above.
- No test suite exists in this repo (`test_logs/` directory exists but looks
  like log output, not pytest — verify before assuming). Everything so far
  has been verified via ad-hoc `.venv/bin/python -c "..."` checks, which is
  consistent with how `extract_attributes.py`/`enhance.py` were validated
  before this session, but there's no `pytest` scaffolding to extend.
- The Go side doesn't know about `merchant_notes` yet — check `kalakriti/`
  for generated proto clients before assuming this is wired end-to-end.

## Quick orientation for a new agent

1. `git status` first if you're about to edit — **this directory is not a
   git repo** as of this writing; don't assume `git diff` will show your
   changes. Confirm before relying on git for anything.
2. Read the specific `§5.x` section of the PDF for whatever RPC you're
   about to implement — don't assume the catalogue table on its own (§5,
   page 7-8) is enough detail; the prose sections (5.A, 5.B and any 5.x
   detail sections) carry the actual constraints.
3. Look at `extract_attributes.py` and `generate_description.py` as the two
   worked examples of "how grounding is enforced structurally, not
   hoped-for" — that's the house pattern, reuse it rather than inventing a
   new anti-hallucination strategy per feature.
4. Never trigger a model download yourself — state the command, stop, let
   the user run it.
5. `scripts/try_*.py` is the fast path to sanity-check a feature without
   spinning up the gRPC server — follow that pattern for new features too.
6. **The codebase drifts between sessions from causes other than your own
   edits** — this exact thing happened: `craft_ontology.json`/`ontology.py`
   were rewritten to schema v2.0.0 by *something* outside any session
   recorded in this file, silently breaking `extract_attributes.py` and
   `generate_description.py`'s `_gi_sentence()` until this session caught
   it. Don't trust this file's own prior "Implemented"/"Verified" claims at
   face value for a module you're about to build on — re-import it and skim
   its current source first; this file records intent and history, not a
   live guarantee that nothing downstream changed since.
7. Every `_model()` should go through `ml_svc.model_loading.resolve_device()`
   and `local_files_only_kwargs()` — see that section above — not a
   hardcoded `device="cpu"` or a bare `from_pretrained(repo_id)`.
