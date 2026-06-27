// Command omnictl is the CLI for omnivoice-core development and operations.
//
// Usage:
//
//	omnictl generate proto    # Generate Go/Python code from proto files
//	omnictl server start      # Start local TTS/STT servers
//	omnictl health            # Check health of local servers
package main

import (
	"os"

	"github.com/plexusone/omnivoice-core/cmd/omnictl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
