#!/bin/bash
# Generate Python proto stubs from omnivoice-core proto definition
#
# Usage: ./generate_proto.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="$SCRIPT_DIR/../../../proto"

if [ ! -d "$PROTO_DIR" ]; then
    echo "Error: Proto directory not found at $PROTO_DIR"
    echo "Expected to be in omnivoice-core/providers/f5tts/server/"
    exit 1
fi

echo "Generating Python proto stubs..."

python3 -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$SCRIPT_DIR" \
    --grpc_python_out="$SCRIPT_DIR" \
    "$PROTO_DIR/localvoice/v1/localvoice.proto"

# Fix imports in generated files (grpc_tools generates absolute imports)
# The generated file imports localvoice_pb2, but we need it to work locally
sed -i '' 's/from localvoice.v1 import localvoice_pb2/import localvoice_pb2/' \
    "$SCRIPT_DIR/localvoice_pb2_grpc.py" 2>/dev/null || true

echo "Generated:"
echo "  - localvoice_pb2.py"
echo "  - localvoice_pb2_grpc.py"
