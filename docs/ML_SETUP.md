# ML setup (ml-svc)

How to get `ml-svc` — the ML gRPC inference service — built and running on a
fresh machine, in both mock mode (default, no weights) and real mode (actual
model weights). For architecture/internals see `services/ml-svc/CLAUDE.md`;
for manual RPC testing see `services/ml-svc/README.md`. This doc is just the
setup/download path.

## Mock mode vs. real mode

`ml-svc`'s Docker image is built with `--build-arg EXTRAS=real` unconditionally
(see `Dockerfile.ml-svc`), so the heavy Python deps (torch, transformers,
rembg, etc.) are always in the image. Whether it actually *uses* them is
controlled purely by the `ML_SVC_MOCK_MODE` env var at runtime:

- **Mock mode** (`docker-compose.yml` default: `ML_SVC_MOCK_MODE: "true"`) —
  every model returns a deterministic fake, no weights are downloaded, no
  GPU/CPU inference happens. This is what `make up`/`make demo-up` give you
  out of the box, and it's enough to exercise the rest of the stack
  end-to-end (bff, core-svc, frontends) without downloading anything.
- **Real mode** (`ML_SVC_MOCK_MODE: "false"`) — actually loads and runs the
  models below. This is what the rest of this doc is about.

## 1. Prerequisites

Same base prerequisites as the rest of the repo (see the root `README.md`'s
Prerequisites table: Go 1.23+, Docker + Compose v2, Python 3.11, `uv`, `make`,
`protoc-gen-go`/`protoc-gen-go-grpc` on `PATH`) — `ml-svc` doesn't add new
host tooling requirements for mock mode. For **real mode** specifically:

- Enough disk space for the weights (see the size table in step 3) plus
  Docker's usual build/image overhead — see the disk-space section of the
  root `CLAUDE.md` if you're on a tight root partition.
- A Hugging Face account + access token if you want Translate (IndicTrans2 is
  gated — see step 6).
- Optionally, an NVIDIA GPU + driver if you want real mode to run at usable
  speed (CPU works, just slow — see step 7).

### Linux (this repo's primary/tested environment)

Nothing beyond the above — Docker Desktop or Docker Engine + Compose v2,
same as the rest of the stack.

### Windows

Run this repo under **WSL2**, not native Windows — the Makefile, shell
scripts (`scripts/gen_proto.sh` etc.), and Docker Compose bind-mount paths
(e.g. `./services/ml-svc/checkpoints/zero_dce_pp`) all assume a POSIX
environment. Install Docker Desktop with the WSL2 backend enabled, clone the
repo *inside* the WSL2 filesystem (e.g. `~/Projects/...`, not `/mnt/c/...`)
for acceptable I/O speed on the venv/model cache, and do everything else
(steps below) from a WSL2 shell. For GPU real mode inside WSL2, install the
NVIDIA driver on the **Windows host** (not inside WSL2 — WSL2 uses the host's
driver via passthrough) plus `nvidia-container-toolkit` inside the WSL2
distro; verify with `nvidia-smi` both on the Windows host and inside WSL2
before trying `docker run --gpus all ...`.

## 2. Bring the service up

```sh
cp .env.example .env      # once, if you haven't already
make up                   # or: docker compose up -d minio minio-init ml-svc
```

`ml-svc` starts in mock mode by default — verify it's serving:

```sh
grpcurl -plaintext localhost:50055 grpc.health.v1.Health/Check   # want {"status":"SERVING"}
```

To switch to real mode, set `ML_SVC_MOCK_MODE: "false"` in `docker-compose.yml`'s
`ml-svc` service (or override it, e.g. `ML_SVC_MOCK_MODE=false docker compose up -d --build ml-svc`),
then follow steps 3–5 below for the weights each RPC needs.

## 3. Where models come from and where they end up

Every real component except two (`image_lighting`, see below, and `vlm` when
`ML_SVC_VLM_BACKEND=llamacpp`, see step 5) is pulled from Hugging Face Hub
**at first use**, not at build time — the first gRPC call that needs a given
model triggers the download, which then persists in the `hf-cache` named
Docker volume (mounted at `/root/.cache/huggingface` in the container, set
via `ML_SVC_MODEL_CACHE_DIR`) so it survives container recreates. Nothing is
pre-baked into the image.

| Component | RPC(s) | Source (HF repo id) | Approx. size | Gated? |
|---|---|---|---|---|
| `vlm`, backend=`transformers` (Qwen2.5-VL, default) | `ExtractAttributes`, `VerifyTechnique`, `GenerateDescription` | `Qwen/Qwen2.5-VL-3B-Instruct` | ~7 GB | No |
| `vlm`, backend=`llamacpp` (same model, GGUF) | same three RPCs, via the `llama-server` sidecar | `ggml-org/Qwen2.5-VL-3B-Instruct-GGUF` | ~2.8 GB (Q4_K_M text + Q8_0 mmproj) | No — see step 5 |
| `image_background` (BiRefNet via rembg) | `EnhanceImage` | `birefnet-general` (rembg's own model registry, downloaded by the `rembg` package, not `transformers`) | ~1 GB | No |
| `embedding` (multilingual e5) | `Embed` | `intfloat/multilingual-e5-base` | small (~1 GB) | No |
| `reranking` (bge reranker) | `Rerank` | `BAAI/bge-reranker-v2-m3` | ~2 GB | No |
| `translation` (IndicTrans2) | not yet wired to an RPC in `server.py`, but its component exists | `ai4bharat/indictrans2-en-indic-1B` | ~1–2 GB | **Yes** — license/contact-info agreement per repo |
| `image_lighting` (Zero-DCE++) | `EnhanceImage` | not on HF Hub — see step 4 | ~2 MB (`.pth` file) | No |
| `voice_transcription` (Bhashini) | `Transcribe` | external HTTP API, no local weights | n/a | Needs API credentials, not a download |
| `handloom_texture` | `DetectHandloom` | no checkpoint exists yet (FFT-only fallback) | n/a | n/a |

**Container storage locations:**
- HF Hub downloads (`vlm`, `embedding`, `reranking`, `translation`) →
  `/root/.cache/huggingface` inside the container, backed by the `hf-cache`
  Docker volume (`docker volume inspect kalakriti_hf-cache` to find the host
  path if you need to poke it directly).
- ONNX-exported copies of `embedding`/`reranking` (sentence-transformers
  doesn't publish/persist these itself for every repo) → also under
  `hf-cache`, at `$HF_HOME/onnx-exports/<repo-id-with-double-dash>/`, so they
  survive restarts without re-exporting.
- `image_background`'s rembg/BiRefNet weights → wherever `rembg` caches by
  default, redirected under the same cache root via a translation
  `app/registry.py`'s `load_all()` performs — see that function's docstring
  for the exact vendor env var (`U2NET_HOME`, etc.) it sets per component
  from `ML_SVC_MODEL_CACHE_DIR`.
- `image_lighting`'s checkpoint → **not** in `hf-cache` at all; it's a
  host-side bind mount, see step 4.

## 4. Zero-DCE++ checkpoint (manual download — not on HF Hub)

`image_lighting`'s one checkpoint has no pip package or HF repo, so it can't
go through the shared `hf-cache` volume like everything else. Fetch it once
per machine:

```sh
mkdir -p services/ml-svc/checkpoints/zero_dce_pp
curl -fLo services/ml-svc/checkpoints/zero_dce_pp/Epoch99.pth \
  https://raw.githubusercontent.com/Li-Chongyi/Zero-DCE_extension/main/Zero-DCE%2B%2B/snapshots_Zero_DCE%2B%2B/Epoch99.pth
sha256sum services/ml-svc/checkpoints/zero_dce_pp/Epoch99.pth
# expect: a4395acb874f320375d9704997cef874eaaaaa26a1777ceb29a92b70f74c3612
```

`services/ml-svc/checkpoints/` is gitignored (it's a binary weights file, not
source) — every fresh clone needs this step run once before real mode is
useful. `docker-compose.yml` bind-mounts the host directory
`./services/ml-svc/checkpoints/zero_dce_pp` read-only to the container path
`/models/zero_dce_pp`, and `ML_SVC_IMAGE_LIGHTING_MODEL` points at the
**container** path (`/models/zero_dce_pp/Epoch99.pth`), not the host path.
If you skip this step, `make up`/mock mode still work fine — Docker
auto-creates the bind-mount source as an empty directory, and the checkpoint
is only read once real mode actually loads `image_lighting`; it degrades to
"unavailable" rather than crashing if missing.

## 5. llama.cpp VLM backend (optional — GGUF checkpoint, not HF Hub via hf-cache)

`vlm` has two backends, picked by `ML_SVC_VLM_BACKEND` (default
`transformers`, the in-process path documented above): `llamacpp` runs the
same Qwen2.5-VL-3B model through a separate `llama-server` container instead
(`services/ml-svc/app/models/vlm/llamacpp.py`), over its OpenAI-compatible
HTTP API. This is the path to use on a small/laptop GPU — see
`services/ml-svc/CLAUDE.md` for why: the `transformers`+bitsandbytes path
stays the better fit for genuinely tiny cards, but llama.cpp is the only one
of the runtimes evaluated that can quantize the vision tower and evict it to
CPU, which matters once VRAM is under ~6GB.

Fetch the two GGUF files once per machine (ungated, no `HF_TOKEN` needed):

```sh
mkdir -p services/ml-svc/checkpoints/llama_vlm
curl -fLo services/ml-svc/checkpoints/llama_vlm/model.gguf \
  https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF/resolve/main/Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf
curl -fLo services/ml-svc/checkpoints/llama_vlm/mmproj.gguf \
  https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF/resolve/main/mmproj-Qwen2.5-VL-3B-Instruct-Q8_0.gguf
```

`services/ml-svc/checkpoints/` is gitignored, same as the Zero-DCE++
checkpoint in step 4 — every fresh clone needs this step run once.
`docker-compose.yml` bind-mounts that directory into the new `llama-server`
service (read-only, container path `/models/llama_vlm`) and starts it with
flags tuned for a ~4GB card (`-c 8192 -np 2 -ngl 99 -ctk q8_0 -ctv q8_0
--image-max-tokens 1024`). Then:

```sh
ML_SVC_VLM_BACKEND=llamacpp ML_SVC_MOCK_MODE=false docker compose up -d --build ml-svc llama-server
```

**If `llama-server` OOMs on a ~4GB card**, edit its `command:` block in
`docker-compose.yml` and add flags in this order (cheapest fix first):
1. `--no-mmproj-offload` — evicts the whole vision tower to CPU RAM, costs
   latency per image, not correctness; frees ~845MB of VRAM.
2. Lower `--image-max-tokens` (e.g. `512`) — bounds the vision-encoder
   activation spike for a large source photo.
3. Lower `-ngl` (fewer transformer layers offloaded to GPU).

**On an 8GB-class card** (e.g. an RTX 4060), there's room to spend on quality
instead of squeezing: swap `Q4_K_M`/`Q8_0` for the `Q8_0` text weights +
`f16` mmproj GGUFs (same HF repo, different filenames — see the repo's file
listing), and raise `-c 65536 -np 8` for real concurrency headroom.

**Before relying on `-np` (concurrent request) serving anywhere**, smoke-test
it yourself: fire 3 concurrent `ExtractAttributes` calls with visibly
different images and confirm no cross-contamination in the responses. This
isn't a known-broken path — llama.cpp's `libmtmd` rewrite gives each server
slot its own multimodal context — but nobody has publicly reconfirmed
multi-slot VLM correctness since that rewrite, so a five-minute check here is
cheaper than debugging a subtle mixed-up-images bug later.

## 6. Gated / credentialed components

**Translate (IndicTrans2)** — every `ai4bharat/indictrans2-*` repo on HF Hub
gates behind a license/contact-info agreement, regardless of checkpoint size.
To use it in real mode:
1. Create a Hugging Face account, visit the
   `ai4bharat/indictrans2-en-indic-1B` model page, and accept the agreement.
2. Generate an access token at https://huggingface.co/settings/tokens.
3. Export it on the host before `docker compose up`:
   ```sh
   export HF_TOKEN=hf_...
   ```
   `docker-compose.yml` reads this into `ML_SVC_TRANSLATION_HF_TOKEN` (`${HF_TOKEN:-}`).
   Every other component works fine without this set.

**Transcribe (Bhashini)** — this is a hosted HTTP API, not a local checkpoint;
set `ML_SVC_VOICE_TRANSCRIPTION_URL`/`ML_SVC_VOICE_TRANSCRIPTION_API_KEY` to
real Bhashini credentials to use it. Not required for anything else in
real mode.

## 7. GPU (optional, Linux and WSL2)

Real mode runs on CPU by default and works fine for everything except the VLM
at a usable speed. To use an NVIDIA GPU:

```sh
docker compose -f docker-compose.yml -f docker-compose.gpu.yml up -d ml-svc
```

Prerequisites (see `docker-compose.gpu.yml`'s header for the full detail):
- NVIDIA driver loaded and `nvidia-smi` working **outside** any container
  (on Windows/WSL2: install the driver on the Windows host, not inside WSL2 —
  WSL2 uses the host driver via passthrough).
- `nvidia-container-toolkit` installed (`pacman -S nvidia-container-toolkit`
  on Arch; see NVIDIA's docs for Ubuntu/WSL2) and
  `nvidia-ctk runtime configure --runtime=docker` run once, then Docker
  restarted.
- Verify with: `docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi`

On a small card (≤4 GB VRAM), `docker-compose.gpu.yml` already pins
`embedding`/`reranking` to CPU (`ML_SVC_TEXT_EMBEDDING_DEVICE`/
`ML_SVC_TEXT_RERANKING_DEVICE`) so the VLM's 4-bit quantization gets the
whole card — remove those two lines if you have 8 GB+ and want all three
sharing the GPU.

## 8. Verifying real mode works

```sh
docker compose logs -f ml-svc   # watch first-call downloads happen
```

The first call to each RPC triggers that component's download — expect the
first `ExtractAttributes`/`VerifyTechnique`/`GenerateDescription` call to take
a while (~7 GB VLM pull), and the first `EnhanceImage` call similarly
(~1 GB BiRefNet pull). Subsequent calls reuse the `hf-cache` volume and are
fast. See `services/ml-svc/README.md` for example `grpcurl` calls for every
RPC, including how to seed a test file into MinIO first (every RPC that
touches media reads bytes from MinIO by `{bucket, object_key}`, never inline).

## 9. Running ml-svc's own test suite (optional, host-side)

Doesn't require Docker or real weights — mock-mode tests are the default and
fast:

```sh
cd services/ml-svc
uv sync --extra dev
./scripts/gen_proto.sh   # generates pb/ — needed before importing app.pb
.venv/bin/python -m pytest
```

To exercise real-mode code paths that don't need actual weights (e.g. "no
checkpoint configured degrades cleanly"), add `--extra real` to `uv sync` —
this pulls the heavy deps onto the host but doesn't download any weights by
itself.
