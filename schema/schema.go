// Package schema provides embedded JSON Schema definitions for OmniVoice formats.
package schema

import (
	_ "embed"
)

// TranscriptV1Schema is the JSON Schema for the OmniVoice Transcript format v1.
//
//go:embed transcript-v1.schema.json
var TranscriptV1Schema string
