"""services/ml-svc/app/models/storage/real.py

MinIO-backed object storage. `minio` is imported here, not at module scope
in `__init__.py`, so mock mode never needs the package installed.
"""

from __future__ import annotations

import io

from app.models.storage import ObjectStorageConfig


class RealObjectStorage:
    def __init__(self, bucket: str, cfg: ObjectStorageConfig) -> None:
        from minio import Minio

        self._bucket = bucket
        self._client = Minio(
            cfg.endpoint.replace("http://", "").replace("https://", ""),
            access_key=cfg.access_key,
            secret_key=cfg.secret_key,
            secure=cfg.endpoint.startswith("https"),
        )

    def get_bytes(self, object_key: str) -> bytes:
        response = self._client.get_object(self._bucket, object_key)
        try:
            return response.read()
        finally:
            response.close()
            response.release_conn()

    def put_bytes(self, object_key: str, data: bytes, content_type: str) -> None:
        self._client.put_object(
            self._bucket, object_key, io.BytesIO(data), len(data), content_type=content_type
        )
