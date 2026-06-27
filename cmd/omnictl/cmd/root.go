package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// rootDir is the omnivoice-core root directory
	rootDir string

	// verbose enables verbose output
	verbose bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "omnictl",
	Short: "CLI for omnivoice-core development and operations",
	Long: `omnictl is a command-line tool for managing omnivoice-core development tasks.

Commands:
  generate proto   Generate Go and Python code from proto files
  server start     Start local TTS/STT servers
  server stop      Stop local servers
  health           Check health of local servers

Examples:
  # Generate all proto files
  omnictl generate proto

  # Start F5-TTS server
  omnictl server start f5tts-mlx

  # Check server health
  omnictl health f5tts-mlx`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&rootDir, "root", "", "omnivoice-core root directory (default: auto-detect)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// initConfig initializes configuration
func initConfig() {
	if rootDir == "" {
		// Auto-detect root directory by looking for go.mod
		rootDir = findRootDir()
	}
}

// findRootDir finds the omnivoice-core root directory by looking for go.mod
func findRootDir() string {
	// Start from current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		gomod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			// Check if this is omnivoice-core by reading go.mod
			data, err := os.ReadFile(gomod)
			if err == nil && contains(string(data), "module github.com/plexusone/omnivoice-core") {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getRootDir returns the root directory, or exits with error if not found
func getRootDir() string {
	if rootDir == "" {
		fmt.Fprintln(os.Stderr, "Error: Could not find omnivoice-core root directory")
		fmt.Fprintln(os.Stderr, "Run from within the omnivoice-core directory or use --root flag")
		os.Exit(1)
	}
	return rootDir
}

// logVerbose prints a message if verbose mode is enabled
func logVerbose(format string, args ...any) {
	if verbose {
		fmt.Printf(format+"\n", args...)
	}
}
