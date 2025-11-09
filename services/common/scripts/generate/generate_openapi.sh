#!/usr/bin/env bash
set -euo pipefail

# Generate Go models and server stubs from OpenAPI specification for API Gateway
# Requires: oapi-codegen (install with: go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest)

ROOT_DIR=$(cd "$(dirname "$0")/../.." && pwd)
SPEC_FILE="$ROOT_DIR/common/wallet-api/openapi.yaml"
API_GATEWAY_DIR="$ROOT_DIR/api-gateway"
MODELS_OUT_DIR="$API_GATEWAY_DIR/internal/models"
SERVER_OUT_DIR="$API_GATEWAY_DIR/internal/server"

echo "Generating Go code from OpenAPI specification..."
echo "Spec file: $SPEC_FILE"
echo "Output directory: $MODELS_OUT_DIR"

# Create output directories if they don't exist
mkdir -p "$MODELS_OUT_DIR"
mkdir -p "$SERVER_OUT_DIR"

# Generate models (types/structs)
echo "Generating models..."
oapi-codegen -package models -generate types \
  -o "$MODELS_OUT_DIR/openapi_models.go" \
  "$SPEC_FILE"

# Generate server interface and handlers
echo "Generating server interface..."
oapi-codegen -package server -generate chi-server \
  -o "$SERVER_OUT_DIR/openapi_server.go" \
  "$SPEC_FILE"

echo "OpenAPI code generation completed successfully!"
echo "Generated files:"
echo "  - Models: $MODELS_OUT_DIR/openapi_models.go"
echo "  - Server: $SERVER_OUT_DIR/openapi_server.go"

exit 0