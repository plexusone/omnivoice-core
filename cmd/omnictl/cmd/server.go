package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	localsttv1 "github.com/plexusone/omnivoice-core/proto/localstt/v1"
	localttsv1 "github.com/plexusone/omnivoice-core/proto/localtts/v1"
)

var (
	// autoLoad automatically loads the model on server start
	autoLoad bool

	// serverModel specifies the model to use (for whisper-mlx)
	serverModel string
)

// Available servers
var availableServers = map[string]struct {
	socket     string
	protoType  string // "tts" or "stt"
	serverFile string
}{
	"f5tts-mlx": {
		socket:     "/tmp/omnivoice-f5tts.sock",
		protoType:  "tts",
		serverFile: "f5tts_server.py",
	},
	"whisper-mlx": {
		socket:     "/tmp/omnivoice-whisper.sock",
		protoType:  "stt",
		serverFile: "whisper_server.py",
	},
}

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage local TTS/STT servers",
	Long:  `Start, stop, and manage local TTS and STT servers.`,
}

// startCmd starts a local server
var startCmd = &cobra.Command{
	Use:   "start <provider>",
	Short: "Start a local server",
	Long: `Start a local TTS or STT server.

Available providers:
  f5tts-mlx    - F5-TTS MLX server for text-to-speech
  whisper-mlx  - Whisper MLX server for speech-to-text

Examples:
  # Start F5-TTS server
  omnictl server start f5tts-mlx

  # Start with auto-load model
  omnictl server start f5tts-mlx --auto-load

  # Start Whisper with specific model
  omnictl server start whisper-mlx --model large-v3-turbo`,
	Args: cobra.ExactArgs(1),
	RunE: runServerStart,
}

// stopCmd stops a local server
var stopCmd = &cobra.Command{
	Use:   "stop <provider>",
	Short: "Stop a local server",
	Long: `Stop a running local server by removing its socket file.

Examples:
  omnictl server stop f5tts-mlx
  omnictl server stop whisper-mlx`,
	Args: cobra.ExactArgs(1),
	RunE: runServerStop,
}

// listCmd lists available servers
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available servers and their status",
	RunE:  runServerList,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(startCmd)
	serverCmd.AddCommand(stopCmd)
	serverCmd.AddCommand(listCmd)

	startCmd.Flags().BoolVar(&autoLoad, "auto-load", false, "automatically load the model on startup")
	startCmd.Flags().StringVar(&serverModel, "model", "", "model to use (for whisper-mlx: tiny, base, small, medium, large-v3-turbo)")
}

func runServerStart(cmd *cobra.Command, args []string) error {
	provider := args[0]

	server, ok := availableServers[provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s\nAvailable: f5tts-mlx, whisper-mlx", provider)
	}

	root := getRootDir()
	serverDir := filepath.Join(root, "providers", provider, "server")

	// Check if server directory exists
	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		return fmt.Errorf("server directory not found: %s", serverDir)
	}

	// Check if socket already exists (server might be running)
	if _, err := os.Stat(server.socket); err == nil {
		return fmt.Errorf("server appears to be running (socket exists: %s)\nRun 'omnictl server stop %s' first", server.socket, provider)
	}

	// Find Python
	pythonCmd := findPythonForServer(serverDir)
	if pythonCmd == "" {
		return fmt.Errorf("Python not found. Create a venv with:\n  cd %s && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt", serverDir)
	}

	// Build command args
	serverScript := filepath.Join(serverDir, server.serverFile)
	cmdArgs := []string{serverScript, "--socket", server.socket}

	if autoLoad {
		cmdArgs = append(cmdArgs, "--auto-load")
	}
	if serverModel != "" && provider == "whisper-mlx" {
		cmdArgs = append(cmdArgs, "--model", serverModel)
	}

	fmt.Printf("Starting %s server...\n", provider)
	fmt.Printf("  Socket: %s\n", server.socket)
	fmt.Printf("  Command: %s %s\n", pythonCmd, serverScript)

	// Start the server
	serverCmd := exec.Command(pythonCmd, cmdArgs...)
	serverCmd.Dir = serverDir
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Printf("\nServer started (PID: %d)\n", serverCmd.Process.Pid)
	fmt.Println("Press Ctrl+C to stop...")

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal or server exit
	done := make(chan error)
	go func() {
		done <- serverCmd.Wait()
	}()

	select {
	case <-sigChan:
		fmt.Println("\nStopping server...")
		_ = serverCmd.Process.Signal(syscall.SIGTERM)
		<-done
	case err := <-done:
		if err != nil {
			return fmt.Errorf("server exited with error: %w", err)
		}
	}

	// Clean up socket
	os.Remove(server.socket)
	fmt.Println("Server stopped.")
	return nil
}

func runServerStop(cmd *cobra.Command, args []string) error {
	provider := args[0]

	server, ok := availableServers[provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	// Remove socket file
	if _, err := os.Stat(server.socket); os.IsNotExist(err) {
		fmt.Printf("Server %s is not running (socket not found)\n", provider)
		return nil
	}

	if err := os.Remove(server.socket); err != nil {
		return fmt.Errorf("failed to remove socket: %w", err)
	}

	fmt.Printf("Removed socket for %s\n", provider)
	fmt.Println("Note: If the server process is still running, it will detect the missing socket and exit.")
	return nil
}

func runServerList(cmd *cobra.Command, args []string) error {
	fmt.Println("Available servers:")
	fmt.Println()

	for name, server := range availableServers {
		status := "stopped"
		details := ""

		// Check if socket exists
		if _, err := os.Stat(server.socket); err == nil {
			// Try to connect and check health
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			healthy, info := checkServerHealth(ctx, server.socket, server.protoType)
			cancel()

			if healthy {
				status = "running"
				details = info
			} else {
				status = "socket exists (not responding)"
			}
		}

		fmt.Printf("  %-15s [%s]\n", name, status)
		fmt.Printf("    Socket: %s\n", server.socket)
		if details != "" {
			fmt.Printf("    %s\n", details)
		}
		fmt.Println()
	}

	return nil
}

// findPythonForServer finds a Python interpreter for the server
func findPythonForServer(serverDir string) string {
	// Try venv first
	venvPython := filepath.Join(serverDir, ".venv", "bin", "python3")
	if _, err := os.Stat(venvPython); err == nil {
		return venvPython
	}

	// Try arch -arm64 for Apple Silicon
	if python, err := exec.LookPath("python3"); err == nil {
		return python
	}

	return ""
}

// checkServerHealth checks if a server is healthy
func checkServerHealth(ctx context.Context, socket, protoType string) (bool, string) {
	conn, err := grpc.NewClient("unix://"+socket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return false, ""
	}
	defer conn.Close()

	switch protoType {
	case "tts":
		client := localttsv1.NewLocalTTSClient(conn)
		resp, err := client.Health(ctx, &localttsv1.HealthRequest{})
		if err != nil {
			return false, ""
		}
		info := fmt.Sprintf("Model: %s (loaded: %v)", resp.ModelName, resp.ModelLoaded)
		return resp.Healthy, info

	case "stt":
		client := localsttv1.NewLocalSTTClient(conn)
		resp, err := client.Health(ctx, &localsttv1.HealthRequest{})
		if err != nil {
			return false, ""
		}
		info := fmt.Sprintf("Model: %s (loaded: %v)", resp.ModelName, resp.ModelLoaded)
		return resp.Healthy, info
	}

	return false, ""
}
