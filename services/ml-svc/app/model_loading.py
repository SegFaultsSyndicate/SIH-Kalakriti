"""services/ml-svc/app/model_loading.py

Shared device/dtype selection and local-cache-first loading, used by every HF
`from_pretrained()` call across the real backends: `app/models/embedding/real.py`,
`app/models/reranking/real.py`, `app/models/vlm/real.py`. Only ever imported
from inside one of those modules' own loading methods -- mock mode must stay
free of torch, so nothing here runs at module import time except cheap,
torch-only setup that real mode already pays for.

DESIGN DECISION: local-cache-first, not "ask the Hub, fall back to disk".
`from_pretrained(repo_id)` without `local_files_only` makes a network call on
every load — an HTTP HEAD/GET against the Hub to check for a newer revision —
even when a complete snapshot already sits under `$HF_HOME`. Models here are
pinned by repo id with no version floating, so that check buys nothing and
costs a real startup dependency on network reachability. `is_cached_locally()`
answers "is this repo id already a complete local entry" via
`huggingface_hub.scan_cache_dir()`; if so, `from_pretrained()` gets
`local_files_only=True` and makes zero network calls. If not, normal
download-on-first-use behaviour applies, unchanged.

DESIGN DECISION: device/dtype prefers CUDA + bfloat16. bfloat16 over float16
because it shares float32's exponent range (no loss-scaling needed) and is
what the Qwen model cards recommend for inference on Ampere+ GPUs. CPU stays
float32.
"""

from __future__ import annotations

import logging
import os
from functools import lru_cache

import torch

log = logging.getLogger("ml-svc.model_loading")


def _configure_cpu_threads() -> None:
    cpu_threads = max(1, min(8, os.cpu_count() or 4))
    torch.set_num_threads(cpu_threads)
    torch.set_num_interop_threads(1)


_configure_cpu_threads()


@lru_cache(maxsize=1)
def resolve_device() -> tuple[str, torch.dtype]:
    """Returns (device, dtype) a `from_pretrained()`/`.to()` call should use."""
    if torch.cuda.is_available():
        return "cuda", torch.bfloat16
    return "cpu", torch.float32


@lru_cache(maxsize=128)
def is_cached_locally(repo_id: str) -> bool:
    """Whether `repo_id` already has a complete snapshot in the local HF cache.

    A repo entry in `scan_cache_dir()` appears the moment *any* file for it
    has downloaded -- e.g. just `config.json`/tokenizer files, which finish in
    milliseconds even when the multi-GB weight shards behind them never did
    (an interrupted load, a crash between resolving the repo and finishing the
    download). Treating that as "cached" forces `local_files_only=True` on a
    snapshot that's missing its weights, and transformers' file-resolution
    doesn't raise a clean error for that -- it hands back `None` for the
    resolved archive path, which crashes deeper in `from_pretrained` with a
    confusing `AttributeError: 'NoneType' object has no attribute 'endswith'`
    instead of the expected `OSError`. So this checks for an actual weight
    file, not just repo presence.
    """
    from huggingface_hub import scan_cache_dir

    try:
        cache_info = scan_cache_dir()
    except Exception:
        log.exception("failed to scan local HF cache for %r; assuming not cached", repo_id)
        return False
    return any(
        repo.repo_id == repo_id
        and any(
            file.file_name.endswith((".safetensors", ".bin"))
            for revision in repo.revisions
            for file in revision.files
        )
        for repo in cache_info.repos
    )


def local_files_only_kwargs(repo_id: str) -> dict[str, bool]:
    """kwargs to splat into from_pretrained(): {"local_files_only": True} once
    `repo_id` is already installed, {} (normal download-if-needed) otherwise."""
    return {"local_files_only": True} if is_cached_locally(repo_id) else {}
