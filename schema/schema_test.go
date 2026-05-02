package schema

import (
	"encoding/json"
	"testing"
)

func TestTranscriptV1Schema_ValidJSON(t *testing.T) {
	if TranscriptV1Schema == "" {
		t.Fatal("TranscriptV1Schema is empty")
	}

	var schema map[string]any
	if err := json.Unmarshal([]byte(TranscriptV1Schema), &schema); err != nil {
		t.Fatalf("TranscriptV1Schema is not valid JSON: %v", err)
	}

	// Verify it's a JSON Schema
	if schemaURI, ok := schema["$schema"].(string); !ok {
		t.Error("TranscriptV1Schema missing $schema field")
	} else if schemaURI == "" {
		t.Error("TranscriptV1Schema $schema field is empty")
	}

	// Verify required fields
	if _, ok := schema["type"]; !ok {
		t.Error("TranscriptV1Schema missing type field")
	}
	if _, ok := schema["properties"]; !ok {
		t.Error("TranscriptV1Schema missing properties field")
	}
}

func TestTranscriptV1Schema_HasExpectedProperties(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(TranscriptV1Schema), &schema); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties is not an object")
	}

	expectedProps := []string{
		"$schema",
		"version",
		"text",
		"language",
		"duration_ms",
		"segments",
		"metadata",
	}

	for _, prop := range expectedProps {
		if _, ok := properties[prop]; !ok {
			t.Errorf("TranscriptV1Schema missing property: %s", prop)
		}
	}
}
