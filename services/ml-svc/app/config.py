"""services/ml-svc/app/config.py

Global configuration from the environment, read once at startup. No model is
touched here; this module is import-safe with no weights on disk.

Only settings shared across the whole service live here: ports, mock mode,
batching knobs, logging, and the two paths/names every feature or component
might reasonably need (the craft ontology allowlist, the media bucket name).
Everything specific to one swappable model/backend -- its repo id, its API
endpoint, its checkpoint path -- lives in that component's own `Config` in
`app/models/<component>/__init__.py`, read via that component's own
`from_env()`. See `app/registry.py` for how the two compose: every component's
`build(cfg)` receives this global `Config` and pulls its own env vars itself.
"""

from __future__ import annotations

import csv
import os
from dataclasses import dataclass
from pathlib import Path

def _find_default_allowlist() -> Path:
    """Walk up from this file looking for the repo's scripts/data/crafts.csv,
    so ml-svc constrains the VLM to exactly the craft codes core-svc seeded
    from the same CSV.

    A full checkout has this file four levels under the repo root
    (services/ml-svc/app/config.py), so the walk finds it immediately. A
    container image that flattens that layout (e.g. Dockerfile's `WORKDIR
    /srv`, which puts this file at /srv/app/config.py with only three real
    parents) has no such ancestor to find -- a fixed `.parents[3]` index used
    to raise IndexError there instead of degrading gracefully. Both
    Dockerfiles set ML_SVC_CRAFT_ALLOWLIST explicitly, so this fallback is
    only ever reached by a container that skipped that, or by a from-scratch
    invocation with no seed CSV at all -- in which case a missing file is not
    fatal (see load_craft_allowlist below), just an empty allowlist.
    """
    here = Path(__file__).resolve()
    for ancestor in here.parents:
        candidate = ancestor / "scripts" / "data" / "crafts.csv"
        if candidate.is_file():
            return candidate
    return Path("scripts/data/crafts.csv")


# Where the seeded ontology lives relative to this file, so ml-svc constrains the
# VLM to exactly the craft codes core-svc seeded from the same CSV.
_DEFAULT_ALLOWLIST = _find_default_allowlist()


def _flag(name: str, default: bool) -> bool:
    return os.getenv(name, str(default)).strip().lower() in {"1", "true", "yes", "on"}


@dataclass(frozen=True)
class Config:
    """Global settings every feature/component may need. Passed as-is into
    every component's `build(cfg)` -- see `app/registry.py`."""

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
    # Where every real component's downloaded weights are cached, regardless
    # of which vendor library backs it. Empty means "let each library use its
    # own default location" (unpersisted across container recreates, but
    # never wrong). Deliberately generic, not named after any one vendor
    # (e.g. not HF_HOME) -- app/registry.py's load_all() is the one place
    # that translates this into whatever env var each library actually reads
    # (HF_HOME for the four huggingface_hub-backed components, U2NET_HOME for
    # rembg), so no vendor-specific name needs to leak out to docker-compose.yml
    # or into any single component's own config.
    model_cache_dir: str = ""

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
            media_bucket=os.getenv("ML_SVC_OBJECT_STORAGE_BUCKET", "kalakriti-media"),
            model_cache_dir=os.getenv("ML_SVC_MODEL_CACHE_DIR", ""),
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


def load_craft_vocab(path: Path) -> dict[str, dict[str, list[str]]]:
    """Per-craft material/technique vocabularies from the same seed CSV.

    crafts.csv carries pipe-separated `techniques` and `materials` columns
    per craft code. Once a craft is known (declared or resolved), the
    extractor should never accept a material/technique for it that isn't in
    this list -- the closed-vocabulary discipline `kalakriti-ml-svc` proved
    out, applied to this service's flat AttributeSet instead of a per-subtype
    schema. A CSV in the plain-newline-list shape `load_craft_allowlist` also
    accepts has no such columns; that's not an error, it just means no craft
    has a vocabulary here, so the extractor falls back to trusting the model's
    raw answer for material/technique (unchanged prior behaviour) rather than
    rejecting everything.
    """
    try:
        rows = csv.DictReader(path.read_text(encoding="utf-8").splitlines())
    except OSError:
        return {}

    vocab: dict[str, dict[str, list[str]]] = {}
    for row in rows:
        code = (row.get("code") or "").strip()
        if not code:
            continue
        vocab[code] = {
            "techniques": [t.strip() for t in (row.get("techniques") or "").split("|") if t.strip()],
            "materials": [m.strip() for m in (row.get("materials") or "").split("|") if m.strip()],
        }
    return vocab
