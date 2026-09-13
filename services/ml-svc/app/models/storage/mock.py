"""services/ml-svc/app/models/storage/mock.py

No network, no disk: reads always return empty bytes, writes are no-ops.
Every component downstream of storage is written to treat empty bytes as
"nothing to process" rather than a decode error, so a mock object round-trip
degrades honestly instead of faking pixel content that nothing actually
inspects. Determinism for mock features that need to vary by object key
(e.g. transcription) comes from the `object_key` argument passed alongside
storage calls, not from anything storage itself returns.
"""

from __future__ import annotations


class MockObjectStorage:
    def get_bytes(self, object_key: str) -> bytes:
        return b""

    def put_bytes(self, object_key: str, data: bytes, content_type: str) -> None:
        return None
