package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// generatePython also generates Python proto files for servers
	generatePython bool

	// cleanFirst removes existing generated files before generating
	cleanFirst bool
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code from proto files",
	Long:  `Generate Go and optionally Python code from proto definition files.`,
}

// protoCmd generates proto code
var protoCmd = &cobra.Command{
	Use:   "proto",
	Short: "Generate Go/Python code from proto files",
	Long: `Generate Go code from proto files using buf or protoc.

This command generates:
  - Go protobuf and gRPC code in proto/localtts/v1/ and proto/localstt/v1/
  - Optionally Python gRPC code for local servers (with --python flag)

Prerequisites:
  - buf (recommended): brew install buf
  - OR protoc with plugins:
      brew install protobuf
      go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
      go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

Examples:
  # Generate Go proto files only
  omnictl generate proto

  # Generate both Go and Python proto files
  omnictl generate proto --python

  # Clean and regenerate
  omnictl generate proto --clean`,
	RunE: runGenerateProto,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(protoCmd)

	protoCmd.Flags().BoolVar(&generatePython, "python", false, "also generate Python proto files for servers")
	protoCmd.Flags().BoolVar(&cleanFirst, "clean", false, "remove existing generated files before generating")
}

func runGenerateProto(cmd *cobra.Command, args []string) error {
	root := getRootDir()
	protoDir := filepath.Join(root, "proto")

	// Check for buf or protoc
	useBuf := commandExists("buf")
	useProtoc := commandExists("protoc")

	if !useBuf && !useProtoc {
		return fmt.Errorf("neither buf nor protoc found. Install with:\n  brew install buf\nOR\n  brew install protobuf")
	}

	// Clean if requested
	if cleanFirst {
		fmt.Println("Cleaning existing generated files...")
		cleanGeneratedFiles(protoDir)
	}

	// Generate Go code
	fmt.Println("Generating Go proto files...")

	if useBuf {
		if err := runBufGenerate(root); err != nil {
			return err
		}
	} else {
		if err := runProtocGenerate(protoDir); err != nil {
			return err
		}
	}

	fmt.Println("  Generated: proto/localtts/v1/localtts.pb.go")
	fmt.Println("  Generated: proto/localtts/v1/localtts_grpc.pb.go")
	fmt.Println("  Generated: proto/localstt/v1/localstt.pb.go")
	fmt.Println("  Generated: proto/localstt/v1/localstt_grpc.pb.go")

	// Generate Python code if requested
	if generatePython {
		fmt.Println("\nGenerating Python proto files...")

		if err := runPythonProtoGenerate(root, "f5tts-mlx", "localtts"); err != nil {
			fmt.Printf("  Warning: Failed to generate Python for f5tts-mlx: %v\n", err)
		} else {
			fmt.Println("  Generated: providers/f5tts-mlx/server/localtts_pb2.py")
			fmt.Println("  Generated: providers/f5tts-mlx/server/localtts_pb2_grpc.py")
		}

		if err := runPythonProtoGenerate(root, "whisper-mlx", "localstt"); err != nil {
			fmt.Printf("  Warning: Failed to generate Python for whisper-mlx: %v\n", err)
		} else {
			fmt.Println("  Generated: providers/whisper-mlx/server/localstt_pb2.py")
			fmt.Println("  Generated: providers/whisper-mlx/server/localstt_pb2_grpc.py")
		}
	}

	fmt.Println("\nProto generation complete!")
	return nil
}

// commandExists checks if a command is available in PATH
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// runBufGenerate runs buf generate
func runBufGenerate(root string) error {
	cmd := exec.Command("buf", "generate", "proto")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logVerbose("Running: buf generate proto (in %s)", root)
	return cmd.Run()
}

// runProtocGenerate runs protoc to generate Go code
func runProtocGenerate(protoDir string) error {
	protos := []struct {
		dir   string
		proto string
	}{
		{"localtts/v1", "localtts.proto"},
		{"localstt/v1", "localstt.proto"},
	}

	for _, p := range protos {
		protoPath := filepath.Join(protoDir, p.dir, p.proto)

		cmd := exec.Command("protoc",
			"--go_out=.", "--go_opt=paths=source_relative",
			"--go-grpc_out=.", "--go-grpc_opt=paths=source_relative",
			"-I", protoDir,
			protoPath,
		)
		cmd.Dir = protoDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		logVerbose("Running: protoc %s", protoPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("protoc failed for %s: %w", p.proto, err)
		}
	}

	return nil
}

// runPythonProtoGenerate generates Python proto files for a provider
func runPythonProtoGenerate(root, provider, protoName string) error {
	serverDir := filepath.Join(root, "providers", provider, "server")
	protoDir := filepath.Join(root, "proto", protoName, "v1")
	protoFile := filepath.Join(protoDir, protoName+".proto")

	// Check if server directory exists
	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return fmt.Errorf("server directory not found: %s", serverDir)
	}

	// Find Python with grpcio-tools
	pythonCmd := findPythonWithGrpc(serverDir)
	if pythonCmd == "" {
		return fmt.Errorf("Python with grpcio-tools not found. Install with: pip install grpcio-tools")
	}

	// Run grpc_tools.protoc
	cmd := exec.Command(pythonCmd, "-m", "grpc_tools.protoc",
		"-I", protoDir,
		"--python_out="+serverDir,
		"--grpc_python_out="+serverDir,
		protoFile,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logVerbose("Running: %s -m grpc_tools.protoc %s", pythonCmd, protoFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("grpc_tools.protoc failed: %w", err)
	}

	// Fix imports in generated grpc file
	grpcFile := filepath.Join(serverDir, protoName+"_pb2_grpc.py")
	if err := fixPythonImports(grpcFile, protoName); err != nil {
		logVerbose("Warning: Failed to fix imports in %s: %v", grpcFile, err)
	}

	return nil
}

// findPythonWithGrpc finds a Python interpreter with grpcio-tools installed
func findPythonWithGrpc(serverDir string) string {
	// Try venv first
	venvPython := filepath.Join(serverDir, ".venv", "bin", "python3")
	if _, err := os.Stat(venvPython); err == nil {
		if checkGrpcTools(venvPython) {
			return venvPython
		}
	}

	// Try system Python
	pythons := []string{"python3", "python"}
	for _, p := range pythons {
		if path, err := exec.LookPath(p); err == nil {
			if checkGrpcTools(path) {
				return path
			}
		}
	}

	return ""
}

// checkGrpcTools checks if Python has grpcio-tools installed
func checkGrpcTools(python string) bool {
	cmd := exec.Command(python, "-c", "import grpc_tools.protoc")
	return cmd.Run() == nil
}

// fixPythonImports fixes the imports in generated Python gRPC files
func fixPythonImports(grpcFile, protoName string) error {
	data, err := os.ReadFile(grpcFile)
	if err != nil {
		return err
	}

	// Replace absolute import with relative import
	oldImport := fmt.Sprintf("from %s.v1 import %s_pb2", protoName, protoName)
	newImport := fmt.Sprintf("import %s_pb2", protoName)

	content := string(data)
	if !containsAt(content, oldImport) {
		return nil // No fix needed
	}

	content = replaceAll(content, oldImport, newImport)
	return os.WriteFile(grpcFile, []byte(content), 0644)
}

// replaceAll replaces all occurrences of old with new in s
func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			result += s
			break
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
	return result
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// cleanGeneratedFiles removes existing generated proto files
func cleanGeneratedFiles(protoDir string) {
	patterns := []string{
		filepath.Join(protoDir, "localtts", "v1", "*.pb.go"),
		filepath.Join(protoDir, "localstt", "v1", "*.pb.go"),
		filepath.Join(protoDir, "localvoice", "v1", "*.pb.go"),
	}

	for _, pattern := range patterns {
		files, _ := filepath.Glob(pattern)
		for _, f := range files {
			logVerbose("Removing: %s", f)
			os.Remove(f)
		}
	}
}
