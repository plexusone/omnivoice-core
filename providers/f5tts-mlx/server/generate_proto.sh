#!/bin/bash
# Generate Python proto stubs from omnivoice-core proto definition
#
# Usage: ./generate_proto.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="$SCRIPT_DIR/../../../proto/localtts/v1"

if [ ! -d "$PROTO_DIR" ]; then
    echo "Error: Proto directory not found at $PROTO_DIR"
    echo "Expected to be in omnivoice-core/providers/f5tts-mlx/server/"
    exit 1
fi

echo "Generating Python proto stubs from localtts.proto..."

# Use venv python if available
PYTHON="${SCRIPT_DIR}/.venv/bin/python3"
if [ ! -f "$PYTHON" ]; then
    PYTHON="python3"
fi

"$PYTHON" -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$SCRIPT_DIR" \
    --grpc_python_out="$SCRIPT_DIR" \
    "$PROTO_DIR/localtts.proto"

# Move generated files from nested structure to flat (if grpc_tools creates nested)
if [ -f "$SCRIPT_DIR/localtts/v1/localtts_pb2.py" ]; then
    mv "$SCRIPT_DIR/localtts/v1/localtts_pb2.py" "$SCRIPT_DIR/"
    mv "$SCRIPT_DIR/localtts/v1/localtts_pb2_grpc.py" "$SCRIPT_DIR/"
    rm -rf "$SCRIPT_DIR/localtts"
fi

# Fix imports in generated files (grpc_tools generates absolute imports)
if [ -f "$SCRIPT_DIR/localtts_pb2_grpc.py" ]; then
    sed -i '' 's/from localtts.v1 import localtts_pb2/import localtts_pb2/' \
        "$SCRIPT_DIR/localtts_pb2_grpc.py" 2>/dev/null || \
    sed -i 's/from localtts.v1 import localtts_pb2/import localtts_pb2/' \
        "$SCRIPT_DIR/localtts_pb2_grpc.py"
fi

# Clean up old localvoice files if they exist
rm -f "$SCRIPT_DIR/localvoice_pb2.py" "$SCRIPT_DIR/localvoice_pb2_grpc.py"

echo "Generated:"
echo "  - localtts_pb2.py"
echo "  - localtts_pb2_grpc.py"
