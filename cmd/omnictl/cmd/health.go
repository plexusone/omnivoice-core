package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	localsttv1 "github.com/plexusone/omnivoice-core/proto/localstt/v1"
	localttsv1 "github.com/plexusone/omnivoice-core/proto/localtts/v1"
)

var (
	// healthTimeout is the timeout for health checks
	healthTimeout time.Duration

	// showRuntime shows runtime info in addition to health
	showRuntime bool
)

// healthCmd checks health of local servers
var healthCmd = &cobra.Command{
	Use:   "health [provider...]",
	Short: "Check health of local servers",
	Long: `Check the health status of local TTS/STT servers.

If no providers are specified, checks all known servers.

Examples:
  # Check all servers
  omnictl health

  # Check specific server
  omnictl health f5tts-mlx

  # Check with runtime info
  omnictl health --runtime`,
	RunE: runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)

	healthCmd.Flags().DurationVar(&healthTimeout, "timeout", 5*time.Second, "timeout for health check")
	healthCmd.Flags().BoolVar(&showRuntime, "runtime", false, "show runtime information")
}

func runHealth(cmd *cobra.Command, args []string) error {
	// Determine which providers to check
	providers := args
	if len(providers) == 0 {
		// Check all providers
		for name := range availableServers {
			providers = append(providers, name)
		}
	}

	hasErrors := false

	for _, provider := range providers {
		server, ok := availableServers[provider]
		if !ok {
			fmt.Printf("Unknown provider: %s\n", provider)
			hasErrors = true
			continue
		}

		fmt.Printf("=== %s ===\n", provider)

		// Check if socket exists
		if _, err := os.Stat(server.socket); os.IsNotExist(err) {
			fmt.Printf("  Status: not running (socket not found)\n")
			fmt.Printf("  Socket: %s\n\n", server.socket)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)

		switch server.protoType {
		case "tts":
			if err := checkTTSHealth(ctx, server.socket); err != nil {
				fmt.Printf("  Status: error - %v\n\n", err)
				hasErrors = true
			}
		case "stt":
			if err := checkSTTHealth(ctx, server.socket); err != nil {
				fmt.Printf("  Status: error - %v\n\n", err)
				hasErrors = true
			}
		}

		cancel()
		fmt.Println()
	}

	if hasErrors {
		return fmt.Errorf("some health checks failed")
	}
	return nil
}

func checkTTSHealth(ctx context.Context, socket string) error {
	conn, err := grpc.NewClient("unix://"+socket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	client := localttsv1.NewLocalTTSClient(conn)

	// Health check
	health, err := client.Health(ctx, &localttsv1.HealthRequest{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	fmt.Printf("  Status: %s\n", boolToStatus(health.Healthy))
	fmt.Printf("  Model loaded: %v\n", health.ModelLoaded)
	if health.ModelName != "" {
		fmt.Printf("  Model: %s (v%s)\n", health.ModelName, health.ModelVersion)
	}
	if len(health.AvailableVoices) > 0 {
		fmt.Printf("  Voices: %v\n", health.AvailableVoices)
	}

	// Runtime info
	if showRuntime {
		runtime, err := client.RuntimeInfo(ctx, &localttsv1.RuntimeInfoRequest{})
		if err != nil {
			fmt.Printf("  Runtime: error - %v\n", err)
		} else {
			fmt.Printf("  Device: %s\n", runtime.DeviceType)
			fmt.Printf("  Framework: %s\n", runtime.FrameworkVersion)
			fmt.Printf("  Python: %s\n", runtime.PythonVersion)
			if runtime.MemoryUsedMb > 0 {
				fmt.Printf("  Memory: %d MB used\n", runtime.MemoryUsedMb)
			}
		}
	}

	return nil
}

func checkSTTHealth(ctx context.Context, socket string) error {
	conn, err := grpc.NewClient("unix://"+socket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	client := localsttv1.NewLocalSTTClient(conn)

	// Health check
	health, err := client.Health(ctx, &localsttv1.HealthRequest{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	fmt.Printf("  Status: %s\n", boolToStatus(health.Healthy))
	fmt.Printf("  Model loaded: %v\n", health.ModelLoaded)
	if health.ModelName != "" {
		fmt.Printf("  Model: %s (v%s)\n", health.ModelName, health.ModelVersion)
	}
	if len(health.SupportedLanguages) > 0 && len(health.SupportedLanguages) <= 10 {
		fmt.Printf("  Languages: %v\n", health.SupportedLanguages)
	} else if len(health.SupportedLanguages) > 10 {
		fmt.Printf("  Languages: %d supported\n", len(health.SupportedLanguages))
	}

	// Runtime info
	if showRuntime {
		runtime, err := client.RuntimeInfo(ctx, &localsttv1.RuntimeInfoRequest{})
		if err != nil {
			fmt.Printf("  Runtime: error - %v\n", err)
		} else {
			fmt.Printf("  Device: %s\n", runtime.DeviceType)
			fmt.Printf("  Framework: %s\n", runtime.FrameworkVersion)
			fmt.Printf("  Python: %s\n", runtime.PythonVersion)
			if runtime.MemoryUsedMb > 0 {
				fmt.Printf("  Memory: %d MB used\n", runtime.MemoryUsedMb)
			}
		}
	}

	return nil
}

func boolToStatus(b bool) string {
	if b {
		return "healthy"
	}
	return "unhealthy"
}
