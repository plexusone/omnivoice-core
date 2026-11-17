package schema

import (
	_ "embed"
	"encoding/json"

	"github.com/invopop/jsonschema"

	"github.com/plexusone/omnivoice-core/stt/verify"
)

// VerifyReportV1Schema is the JSON Schema for a verify report, generated
// from verify.Report by `go run schema/gen/main.go`.
//
//go:embed verify-report-v1.schema.json
var VerifyReportV1Schema string

// VerifyReportV1SchemaID is the schema's $id.
const VerifyReportV1SchemaID = "https://omnivoice.dev/schema/verify-report-v1.json"

// GenerateVerifyReportV1 generates the verify report schema from its Go
// types, indented and ending with a newline.
func GenerateVerifyReportV1() ([]byte, error) {
	r := jsonschema.Reflector{RequiredFromJSONSchemaTags: false}
	s := r.Reflect(&verify.Report{})
	s.ID = VerifyReportV1SchemaID
	s.Title = "OmniVoice verify report"
	s.Description = "Quotes checked against a recording by comparing what several speech-to-text engines heard."
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
