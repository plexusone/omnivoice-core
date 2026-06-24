# Proto generation for omnivoice-core
#
# Prerequisites:
#   brew install buf
#   OR
#   brew install protobuf
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: generate clean lint

# Generate Go code from proto files using buf
# Run from omnivoice-core root: buf generate proto
# Or from proto/: make generate
generate:
	@echo "Generating Go code from proto files..."
	cd .. && buf generate proto

# Alternative: Generate using protoc directly
generate-protoc:
	@echo "Generating Go code from proto files using protoc..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		localvoice/v1/localvoice.proto

# Lint proto files
lint:
	buf lint

# Clean generated files
clean:
	rm -f localvoice/v1/*.pb.go

# Format proto files
format:
	buf format -w
