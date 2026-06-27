#!/bin/bash
# Start the F5-TTS MLX gRPC server
#
# Usage:
#   ./run.sh              # Start server (model loaded on first request)
#   ./run.sh --auto-load  # Start server with model pre-loaded

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check if proto stubs exist
if [ ! -f "localvoice_pb2.py" ]; then
    echo "Proto stubs not found. Generating..."
    ./generate_proto.sh
fi

# Check for required packages
python3 -c "import grpc" 2>/dev/null || {
    echo "grpcio not found. Installing dependencies..."
    pip install -r requirements.txt
}

# Start the server
echo "Starting F5-TTS MLX server..."
python3 f5tts_server.py "$@"
