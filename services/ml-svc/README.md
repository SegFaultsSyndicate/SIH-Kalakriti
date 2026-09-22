# ml-svc — manual testing

For architecture/internals see `CLAUDE.md`. This is just "how do I poke it."

## Ports

| Port | What | Notes |
|---|---|---|
| `50055` | gRPC API (`inference.v1.InferenceService`) | reflection enabled — no `.proto` needed for `grpcurl` |
| `9097` → `9095` (container) | Prometheus metrics | host port 9097, not 9095 (bff reserves 9095 for its own, unimplemented, metrics) |
| `9000` | MinIO S3 API | every media-touching RPC fetches image/audio bytes from here |
| `9001` | MinIO web console | `minioadmin` / `minioadmin` |

Bring up just what you need for manual testing:
```sh
docker compose up -d minio minio-init ml-svc
# minio-init is one-shot (creates the `kalakriti` bucket), exits immediately — normal.

# GPU (quantizes the VLM to 4-bit, needs nvidia-container-toolkit — see docker-compose.gpu.yml's header):
docker compose -f docker-compose.yml -f docker-compose.gpu.yml up -d ml-svc
```

Check it's actually serving:
```sh
grpcurl -plaintext localhost:50055 grpc.health.v1.Health/Check   # want {"status":"SERVING"}
```
`ML_SVC_MOCK_MODE` (docker-compose.yml) controls mock vs. real models — mock returns deterministic fakes instantly; real downloads/runs actual weights. `real` mode also needs a few env vars for gated/large parts — see `CLAUDE.md`'s component table, and `ML_SVC_TRANSLATION_HF_TOKEN` for Translate specifically.

## Uploading a test file to MinIO

Every RPC below that takes a `MediaRef` needs the file already sitting in MinIO — gRPC requests carry `{bucket, object_key}`, never raw bytes.

**No install needed** — web console: open `http://localhost:9001` (`minioadmin`/`minioadmin`), open the `kalakriti` bucket, drag your file in.

**CLI**: `sudo pacman -S minio-client` — installs the binary as **`mcli`**, not `mc` (Arch's package avoids clashing with Midnight Commander's `mc`). Then:
```sh
mcli alias set local http://localhost:9000 minioadmin minioadmin
mcli cp your-image.jpg local/kalakriti/test/your-image.jpg
```

**Via the container itself** (already has the `minio` Python package in real mode, nothing extra to install):
```sh
docker cp your-image.jpg kalakriti-ml-svc-1:/tmp/your-image.jpg
docker compose exec -T ml-svc python -c "
from minio import Minio
c = Minio('minio:9000', access_key='minioadmin', secret_key='minioadmin', secure=False)
c.fput_object('kalakriti', 'test/your-image.jpg', '/tmp/your-image.jpg', content_type='image/jpeg')
"
```

## The 9 RPCs

Install `grpcurl`: `sudo pacman -S grpcurl` (or `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`).

### Embed — text only, no MinIO
```sh
grpcurl -plaintext -d '{"texts":["hand-woven Bandhani silk saree"]}' \
  localhost:50055 inference.v1.InferenceService/Embed
```

### Rerank
```sh
grpcurl -plaintext -d '{
  "query":"red silk saree with mirror work",
  "candidates":[
    {"id":"a","text":"Bandhani silk saree with mirror embroidery"},
    {"id":"b","text":"brass Dokra figurine"}
  ]
}' localhost:50055 inference.v1.InferenceService/Rerank
```

### Translate — needs `ML_SVC_TRANSLATION_HF_TOKEN` (IndicTrans2 is a gated HF repo)
```sh
grpcurl -plaintext -d '{
  "title":"Handwoven silk saree",
  "description":"Made using the Bandhani tie-dye technique.",
  "highlights":["100% silk","natural dyes"],
  "source_language":"LANGUAGE_ENGLISH",
  "target_language":"LANGUAGE_HINDI"
}' localhost:50055 inference.v1.InferenceService/Translate
```

### DetectHandloom — needs a weave *close-up*, not a general product photo
```sh
grpcurl -plaintext -d '{
  "media":{"bucket":"kalakriti","object_key":"test/your-image.jpg","mime_type":"image/jpeg"}
}' localhost:50055 inference.v1.InferenceService/DetectHandloom
```

### ExtractAttributes — first call downloads the VLM (~7GB, Qwen2.5-VL-3B)
```sh
grpcurl -plaintext -d '{
  "media":[{"bucket":"kalakriti","object_key":"test/your-image.jpg","mime_type":"image/jpeg"}]
}' localhost:50055 inference.v1.InferenceService/ExtractAttributes
```

### VerifyTechnique — `craft_id` must be a real code from `scripts/data/crafts.csv`
```sh
grpcurl -plaintext -d '{
  "media":[{"bucket":"kalakriti","object_key":"test/your-image.jpg","mime_type":"image/jpeg"}],
  "claimed_technique":"wheel-throwing",
  "craft_id":"pottery"
}' localhost:50055 inference.v1.InferenceService/VerifyTechnique
```

### EnhanceImage — first call downloads BiRefNet (~1GB)
```sh
grpcurl -plaintext -d '{
  "source":{"bucket":"kalakriti","object_key":"test/your-image.jpg","mime_type":"image/jpeg"},
  "remove_background":true,
  "auto_white_balance":true
}' localhost:50055 inference.v1.InferenceService/EnhanceImage
```

### GenerateDescription — feed it a real `attributes` object (e.g. paste back `ExtractAttributes`' response)
```sh
grpcurl -plaintext -d '{
  "attributes":{
    "craft_id":{"value":"pottery","confidence":1.0},
    "material":{"value":"clay","confidence":1.0},
    "colours":{"values":["brown","beige"],"confidence":1.0}
  },
  "craft_id":"pottery",
  "language":"LANGUAGE_ENGLISH"
}' localhost:50055 inference.v1.InferenceService/GenerateDescription
```

### Transcribe — streaming; needs `ML_SVC_VOICE_TRANSCRIPTION_URL`/`_API_KEY` (Bhashini)
```sh
grpcurl -plaintext -d '{
  "audio":{"bucket":"kalakriti","object_key":"test/your-audio.wav","mime_type":"audio/wav"},
  "language":"LANGUAGE_HINDI"
}' localhost:50055 inference.v1.InferenceService/Transcribe
```

## Reference

- `Language` enum values: `proto/common/v1/common.proto` (`LANGUAGE_ENGLISH`, `LANGUAGE_HINDI`, `LANGUAGE_ASSAMESE`, ...).
- Valid `craft_id`s: first column of `scripts/data/crafts.csv` (e.g. `pottery`, `blue-pottery`).
- On a small GPU (≤4GB), see `docker-compose.gpu.yml`'s header — it pins `embedding`/`reranking` to CPU (`ML_SVC_TEXT_EMBEDDING_DEVICE`/`ML_SVC_TEXT_RERANKING_DEVICE`) so the VLM's 4-bit quantization has the whole card.
