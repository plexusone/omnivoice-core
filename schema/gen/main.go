//go:build ignore

// Command gen writes the verify report schema from its Go types.
//
//	go run schema/gen/main.go
package main

import (
	"log"
	"os"

	"github.com/plexusone/omnivoice-core/schema"
)

func main() {
	b, err := schema.GenerateVerifyReportV1()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("schema/verify-report-v1.schema.json", b, 0o600); err != nil {
		log.Fatal(err)
	}
}
