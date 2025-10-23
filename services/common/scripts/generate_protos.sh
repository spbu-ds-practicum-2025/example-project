#!/usr/bin/env bash
set -euo pipefail

# Regenerate protobuf code for Go services (and optionally Python for analytics)
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc on PATH

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
PROTO_ROOT="$ROOT_DIR/services/common"
OUT_DIR="$ROOT_DIR/services"

echo "PROTO_ROOT: $PROTO_ROOT"

echo "Generating Go stubs for bank-service..."
protoc -I="$PROTO_ROOT" \
  --go_out="$OUT_DIR" --go-grpc_out="$OUT_DIR" \
  "$PROTO_ROOT/bank-service-api/bank_service.proto"

echo "Generating Go stubs for bank-card-adapter..."
protoc -I="$PROTO_ROOT" \
  --go_out="$OUT_DIR" --go-grpc_out="$OUT_DIR" \
  "$PROTO_ROOT/bank-card-adapter-api/bank_card_adapter.proto"

echo "Generating Go stubs for analytics-service proto (if used by Go callers)..."
protoc -I="$PROTO_ROOT" \
  --go_out="$OUT_DIR" --go-grpc_out="$OUT_DIR" \
  "$PROTO_ROOT/analytics-service-api/analytics_service.proto"

# Optional: generate Python stubs for analytics-service if analytics will use gRPC directly
# Uncomment and ensure python grpc tools are installed if needed
# echo "Generating Python stubs for analytics-service..."
# protoc -I="$PROTO_ROOT" --python_out="$ROOT_DIR/services/analytics-service/proto" --grpc_python_out="$ROOT_DIR/services/analytics-service/proto" "$PROTO_ROOT/analytics-service-api/analytics_service.proto"

echo "Protos generated (Go). If you need Python stubs for analytics, uncomment the python generation lines above."

exit 0
