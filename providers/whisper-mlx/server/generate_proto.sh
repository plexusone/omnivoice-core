#!/bin/bash
# Generate Python gRPC code from localstt.proto

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="$SCRIPT_DIR/../../../proto/localstt/v1"

echo "Generating Python gRPC code from localstt.proto..."

python -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$SCRIPT_DIR" \
    --grpc_python_out="$SCRIPT_DIR" \
    "$PROTO_DIR/localstt.proto"

# Fix imports in generated files (grpc_tools generates absolute imports)
# Move generated files from nested structure to flat
if [ -f "$SCRIPT_DIR/localstt/v1/localstt_pb2.py" ]; then
    mv "$SCRIPT_DIR/localstt/v1/localstt_pb2.py" "$SCRIPT_DIR/"
    mv "$SCRIPT_DIR/localstt/v1/localstt_pb2_grpc.py" "$SCRIPT_DIR/"
    rm -rf "$SCRIPT_DIR/localstt"
fi

# Fix the import in the grpc file
if [ -f "$SCRIPT_DIR/localstt_pb2_grpc.py" ]; then
    sed -i '' 's/from localstt.v1 import localstt_pb2/import localstt_pb2/' "$SCRIPT_DIR/localstt_pb2_grpc.py" 2>/dev/null || \
    sed -i 's/from localstt.v1 import localstt_pb2/import localstt_pb2/' "$SCRIPT_DIR/localstt_pb2_grpc.py"
fi

echo "Generated:"
ls -la "$SCRIPT_DIR"/*.py 2>/dev/null || echo "No Python files generated"
