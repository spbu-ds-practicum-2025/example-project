#!/usr/bin/env bash
set -euo pipefail

# Regenerate protobuf code for Go services (and optionally Python for analytics)
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc on PATH

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
PROTO_ROOT="$ROOT_DIR/common"
OUT_DIR="$ROOT_DIR"

echo "PROTO_ROOT: $PROTO_ROOT"

echo "Generating Go stubs for bank-service inside bank-service..."
protoc -I="$PROTO_ROOT" \
  --go_out="$OUT_DIR/bank-service/proto" --go-grpc_out="$OUT_DIR/bank-service/proto" \
  "$PROTO_ROOT/bank-service-api/bank_service.proto"

echo "Generating Go stubs for bank-service inside api-gateway..."
protoc -I="$PROTO_ROOT" \
  --go_out="$OUT_DIR/api-gateway/proto" --go-grpc_out="$OUT_DIR/api-gateway/proto" \
  "$PROTO_ROOT/bank-service-api/bank_service.proto"

exit 0
