#!/usr/bin/env bash
# services/ml-svc/scripts/gen_proto.sh
#
# Regenerates pb/**/*_pb2*.py from the repo-root proto/ directory that the Go
# side also generates from (see the root Makefile's `proto` target and
# buf.gen.yaml, which only wires Go plugins). pb/ is gitignored except for
# pb/__init__.py (see .gitignore) — it is a build artifact, run this after
# every proto change and before running the ml-svc test suite or building its
# Docker image.
#
# Requires grpcio-tools, which is in this project's `dev` extra:
#   cd services/ml-svc && uv sync --extra dev && ./scripts/gen_proto.sh
set -euo pipefail
cd "$(dirname "$0")/../../.."  # repo root

PB_OUT=services/ml-svc/pb

rm -rf "$PB_OUT/common" "$PB_OUT/inference"
mkdir -p "$PB_OUT/common/v1" "$PB_OUT/inference/v1"
touch "$PB_OUT/common/__init__.py" "$PB_OUT/common/v1/__init__.py"
touch "$PB_OUT/inference/__init__.py" "$PB_OUT/inference/v1/__init__.py"

python -m grpc_tools.protoc \
  -I proto \
  --python_out="$PB_OUT" \
  --grpc_python_out="$PB_OUT" \
  --pyi_out="$PB_OUT" \
  proto/common/v1/common.proto \
  proto/inference/v1/inference.proto

echo "Generated $PB_OUT/common/v1 and $PB_OUT/inference/v1"
