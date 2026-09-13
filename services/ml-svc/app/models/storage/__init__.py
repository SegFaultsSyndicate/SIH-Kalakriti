"""services/ml-svc/app/models/storage/

The object storage component: fetches and writes the raw bytes every image-
and audio-touching feature needs. Swappable independently of every other
model -- real mode talks to MinIO/S3, mock mode never touches the network.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Protocol

from app.config import Config


class ObjectStorage(Protocol):
    """Plain bytes in, plain bytes out. No PIL, no protobuf: decoding into an
    image (or audio) happens in whichever component actually needs pixels."""

    def get_bytes(self, object_key: str) -> bytes: ...

    def put_bytes(self, object_key: str, data: bytes, content_type: str) -> None: ...


@dataclass(frozen=True)
class ObjectStorageConfig:
    """Real-mode-only connection settings. Empty defaults are fine in mock
    mode, where none of this is read."""

    endpoint: str = ""
    access_key: str = ""
    secret_key: str = ""

    @classmethod
    def from_env(cls) -> "ObjectStorageConfig":
        return cls(
            endpoint=os.getenv("ML_SVC_OBJECT_STORAGE_ENDPOINT", ""),
            access_key=os.getenv("ML_SVC_OBJECT_STORAGE_ACCESS_KEY", ""),
            secret_key=os.getenv("ML_SVC_OBJECT_STORAGE_SECRET_KEY", ""),
        )


def build(cfg: Config) -> ObjectStorage:
    if cfg.mock_mode:
        from app.models.storage.mock import MockObjectStorage

        return MockObjectStorage()

    from app.models.storage.real import RealObjectStorage

    return RealObjectStorage(cfg.media_bucket, ObjectStorageConfig.from_env())
