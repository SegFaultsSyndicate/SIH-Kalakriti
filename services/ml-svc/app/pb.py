"""services/ml-svc/app/pb.py

One place to import generated protobuf from. `make proto` writes protoc's output
into pb/, where the modules import each other by proto path ("from common.v1
import common_pb2"), so pb/ has to be on sys.path rather than imported as a
package. That is the whole reason this module exists.
"""

from __future__ import annotations

import sys
from pathlib import Path

_PB_ROOT = Path(__file__).resolve().parent.parent / "pb"
if str(_PB_ROOT) not in sys.path:
    sys.path.insert(0, str(_PB_ROOT))

from common.v1 import common_pb2  # noqa: E402
from inference.v1 import inference_pb2, inference_pb2_grpc  # noqa: E402

__all__ = ["common_pb2", "inference_pb2", "inference_pb2_grpc"]
