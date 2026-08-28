"""services/ml-svc/app/config.py

Configuration from the environment, read once at startup. No model is touched
here; this module is import-safe with no weights on disk.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

# Where the seeded ontology lives relative to this file, so ml-svc constrains the
# VLM to exactly the craft codes core-svc seeded from the same CSV.
_DEFAULT_ALLOWLIST = Path(__file__).resolve().parents[3] / "scripts" / "data" / "crafts.csv"


def _flag(name: str, default: bool) -> bool:
    return os.getenv(name, str(default)).strip().lower() in {"1", "true", "yes", "on"}


@dataclass(frozen=True)
class Config:
    """Everything the service reads from the environment."""

    grpc_port: int = 50055
    metrics_port: int = 9095
    # Mock mode is the default on purpose: the Go side must never be blocked
    # waiting for weights. Set ML_SVC_MOCK_MODE=false to load real models.
    mock_mode: bool = True
    max_batch_size: int = 16
    max_wait_ms: int = 50
    log_level: str = "INFO"
    model_version: str = "mock-v1"
    craft_allowlist_path: Path = _DEFAULT_ALLOWLIST
    media_bucket: str = "kalakriti-media"
    # Real-mode only; unused and unvalidated in mock mode.
    s3_endpoint: str = ""
    s3_access_key: str = ""
    s3_secret_key: str = ""
    bhashini_url: str = ""
    bhashini_api_key: str = ""
    vlm_model: str = "Qwen/Qwen2.5-VL-3B-Instruct"
    embed_model: str = "intfloat/multilingual-e5-base"
    rerank_model: str = "BAAI/bge-reranker-v2-m3"
    video_frames: int = 8

    @classmethod
    def from_env(cls) -> "Config":
        mock = _flag("ML_SVC_MOCK_MODE", True)
        return cls(
            grpc_port=int(os.getenv("ML_SVC_GRPC_PORT", "50055")),
            metrics_port=int(os.getenv("ML_SVC_METRICS_PORT", "9095")),
            mock_mode=mock,
            max_batch_size=int(os.getenv("ML_SVC_MAX_BATCH_SIZE", "16")),
            max_wait_ms=int(os.getenv("ML_SVC_MAX_WAIT_MS", "50")),
            log_level=os.getenv("LOG_LEVEL", "INFO").upper(),
            model_version=os.getenv("ML_SVC_MODEL_VERSION", "mock-v1" if mock else "kalakriti-2026.09"),
            craft_allowlist_path=Path(os.getenv("ML_SVC_CRAFT_ALLOWLIST", str(_DEFAULT_ALLOWLIST))),
            media_bucket=os.getenv("S3_BUCKET", "kalakriti-media"),
            s3_endpoint=os.getenv("S3_ENDPOINT", ""),
            s3_access_key=os.getenv("S3_ACCESS_KEY", ""),
            s3_secret_key=os.getenv("S3_SECRET_KEY", ""),
            bhashini_url=os.getenv("BHASHINI_ASR_URL", ""),
            bhashini_api_key=os.getenv("BHASHINI_API_KEY", ""),
            vlm_model=os.getenv("ML_SVC_VLM_MODEL", "Qwen/Qwen2.5-VL-3B-Instruct"),
            embed_model=os.getenv("ML_SVC_EMBED_MODEL", "intfloat/multilingual-e5-base"),
            rerank_model=os.getenv("ML_SVC_RERANK_MODEL", "BAAI/bge-reranker-v2-m3"),
            video_frames=int(os.getenv("ML_SVC_VIDEO_FRAMES", "8")),
        )


def load_craft_allowlist(path: Path) -> list[str]:
    """Craft codes the extractor may return, read from the seed CSV core-svc uses.

    Accepts either that CSV (first column is the code) or a plain newline list.
    A missing file is not fatal: it means an empty allowlist, and the extractor
    then abstains rather than inventing a craft.
    """
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError:
        return []

    codes = []
    for i, line in enumerate(lines):
        code = line.split(",", 1)[0].strip()
        if not code or (i == 0 and code == "code"):
            continue
        codes.append(code)
    return codes
