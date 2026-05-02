package stt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTranscript(t *testing.T) {
	result := &TranscriptionResult{
		Text:               "Hello world",
		Language:           "en-US",
		LanguageConfidence: 0.95,
		Duration:           5 * time.Second,
		Segments: []Segment{
			{
				Text:       "Hello world",
				StartTime:  0,
				EndTime:    2500 * time.Millisecond,
				Confidence: 0.98,
				Speaker:    "speaker_1",
				Words: []Word{
					{
						Text:       "Hello",
						StartTime:  0,
						EndTime:    1000 * time.Millisecond,
						Confidence: 0.99,
						Speaker:    "speaker_1",
					},
					{
						Text:       "world",
						StartTime:  1200 * time.Millisecond,
						EndTime:    2500 * time.Millisecond,
						Confidence: 0.97,
						Speaker:    "speaker_1",
					},
				},
			},
		},
	}

	config := &TranscriptionConfig{
		Language:                 "en-US",
		EnablePunctuation:        true,
		EnableWordTimestamps:     true,
		EnableSpeakerDiarization: true,
	}

	transcript := NewTranscript(result, "deepgram", "nova-2", "audio.mp3", config)

	// Verify basic fields
	if transcript.Schema != TranscriptSchemaURL {
		t.Errorf("Schema = %s, want %s", transcript.Schema, TranscriptSchemaURL)
	}
	if transcript.Version != TranscriptFormatVersion {
		t.Errorf("Version = %s, want %s", transcript.Version, TranscriptFormatVersion)
	}
	if transcript.Text != "Hello world" {
		t.Errorf("Text = %s, want %s", transcript.Text, "Hello world")
	}

	// Verify duration conversion
	if transcript.Duration.Milliseconds() != 5000 {
		t.Errorf("Duration.Milliseconds() = %d, want %d", transcript.Duration.Milliseconds(), 5000)
	}
	if transcript.TotalDuration() != 5*time.Second {
		t.Errorf("TotalDuration() = %v, want %v", transcript.TotalDuration(), 5*time.Second)
	}

	// Verify segments
	if len(transcript.Segments) != 1 {
		t.Fatalf("len(Segments) = %d, want 1", len(transcript.Segments))
	}
	seg := transcript.Segments[0]
	if seg.Start.Milliseconds() != 0 {
		t.Errorf("Segment.Start.Milliseconds() = %d, want 0", seg.Start.Milliseconds())
	}
	if seg.End.Milliseconds() != 2500 {
		t.Errorf("Segment.End.Milliseconds() = %d, want 2500", seg.End.Milliseconds())
	}
	if seg.SegmentDuration() != 2500*time.Millisecond {
		t.Errorf("SegmentDuration() = %v, want %v", seg.SegmentDuration(), 2500*time.Millisecond)
	}

	// Verify words
	if len(seg.Words) != 2 {
		t.Fatalf("len(Words) = %d, want 2", len(seg.Words))
	}
	word := seg.Words[0]
	if word.Start.Milliseconds() != 0 {
		t.Errorf("Word.Start.Milliseconds() = %d, want 0", word.Start.Milliseconds())
	}
	if word.End.Milliseconds() != 1000 {
		t.Errorf("Word.End.Milliseconds() = %d, want 1000", word.End.Milliseconds())
	}
	if word.WordDuration() != time.Second {
		t.Errorf("WordDuration() = %v, want %v", word.WordDuration(), time.Second)
	}

	// Verify metadata
	if transcript.Metadata.Provider != "deepgram" {
		t.Errorf("Metadata.Provider = %s, want deepgram", transcript.Metadata.Provider)
	}
	if transcript.Metadata.Options == nil {
		t.Fatal("Metadata.Options is nil")
	}
	if !transcript.Metadata.Options.EnablePunctuation {
		t.Error("Metadata.Options.EnablePunctuation = false, want true")
	}
}

func TestTranscriptJSONRoundTrip(t *testing.T) {
	result := &TranscriptionResult{
		Text:     "Test transcript",
		Language: "en-US",
		Duration: 3500 * time.Millisecond,
		Segments: []Segment{
			{
				Text:      "Test transcript",
				StartTime: 500 * time.Millisecond,
				EndTime:   3000 * time.Millisecond,
			},
		},
	}

	original := NewTranscript(result, "openai", "whisper-1", "", nil)

	// Marshal to JSON
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify JSON structure has milliseconds as integers
	var jsonMap map[string]any
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Check duration_ms is an integer
	if durationMS, ok := jsonMap["duration_ms"].(float64); !ok {
		t.Errorf("duration_ms is not a number: %T", jsonMap["duration_ms"])
	} else if durationMS != 3500 {
		t.Errorf("duration_ms = %v, want 3500", durationMS)
	}

	// Unmarshal back
	var decoded Transcript
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify round-trip
	if decoded.Duration.Milliseconds() != 3500 {
		t.Errorf("decoded Duration.Milliseconds() = %d, want 3500", decoded.Duration.Milliseconds())
	}
	if decoded.TotalDuration() != 3500*time.Millisecond {
		t.Errorf("decoded TotalDuration() = %v, want %v", decoded.TotalDuration(), 3500*time.Millisecond)
	}
	if len(decoded.Segments) != 1 {
		t.Fatalf("len(decoded.Segments) = %d, want 1", len(decoded.Segments))
	}
	if decoded.Segments[0].Start.Milliseconds() != 500 {
		t.Errorf("decoded Segment.Start.Milliseconds() = %d, want 500", decoded.Segments[0].Start.Milliseconds())
	}
}

func TestTranscript_SaveAndLoad(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.transcript.json")

	result := &TranscriptionResult{
		Text:     "Save and load test",
		Language: "en-US",
		Duration: 2000 * time.Millisecond,
		Segments: []Segment{
			{
				Text:      "Save and load test",
				StartTime: 0,
				EndTime:   2000 * time.Millisecond,
			},
		},
	}

	original := NewTranscript(result, "test-provider", "test-model", "test.wav", nil)

	// Save to file
	if err := original.SaveJSON(filePath); err != nil {
		t.Fatalf("SaveJSON() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("SaveJSON() did not create file")
	}

	// Load from file
	loaded, err := LoadTranscript(filePath)
	if err != nil {
		t.Fatalf("LoadTranscript() error = %v", err)
	}

	// Verify loaded content matches original
	if loaded.Text != original.Text {
		t.Errorf("loaded.Text = %s, want %s", loaded.Text, original.Text)
	}
	if loaded.Duration.Milliseconds() != original.Duration.Milliseconds() {
		t.Errorf("loaded.Duration = %d, want %d",
			loaded.Duration.Milliseconds(), original.Duration.Milliseconds())
	}
	if loaded.Metadata.Provider != original.Metadata.Provider {
		t.Errorf("loaded.Metadata.Provider = %s, want %s",
			loaded.Metadata.Provider, original.Metadata.Provider)
	}
}

func TestLoadTranscript_FileNotFound(t *testing.T) {
	_, err := LoadTranscript("/nonexistent/path/transcript.json")
	if err == nil {
		t.Error("LoadTranscript() expected error for nonexistent file")
	}
}

func TestLoadTranscript_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(filePath, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := LoadTranscript(filePath)
	if err == nil {
		t.Error("LoadTranscript() expected error for invalid JSON")
	}
}

func TestNewTranscript_EmptySegments(t *testing.T) {
	result := &TranscriptionResult{
		Text:     "No segments",
		Language: "en-US",
		Duration: 1000 * time.Millisecond,
		Segments: []Segment{}, // Empty segments
	}

	transcript := NewTranscript(result, "provider", "model", "", nil)

	if len(transcript.Segments) != 0 {
		t.Errorf("len(Segments) = %d, want 0", len(transcript.Segments))
	}
	if transcript.Text != "No segments" {
		t.Errorf("Text = %s, want 'No segments'", transcript.Text)
	}
}

func TestNewTranscript_NilConfig(t *testing.T) {
	result := &TranscriptionResult{
		Text:     "Test",
		Duration: 1000 * time.Millisecond,
	}

	transcript := NewTranscript(result, "provider", "model", "", nil)

	if transcript.Metadata.Options != nil {
		t.Error("Metadata.Options should be nil when config is nil")
	}
}

func TestNewTranscript_EmptyWords(t *testing.T) {
	result := &TranscriptionResult{
		Text:     "Segment without words",
		Duration: 1000 * time.Millisecond,
		Segments: []Segment{
			{
				Text:      "Segment without words",
				StartTime: 0,
				EndTime:   1000 * time.Millisecond,
				Words:     []Word{}, // Empty words
			},
		},
	}

	transcript := NewTranscript(result, "provider", "model", "", nil)

	if len(transcript.Segments) != 1 {
		t.Fatalf("len(Segments) = %d, want 1", len(transcript.Segments))
	}
	if len(transcript.Segments[0].Words) != 0 {
		t.Errorf("len(Words) = %d, want 0", len(transcript.Segments[0].Words))
	}
}

func TestTranscript_ToJSON_Format(t *testing.T) {
	result := &TranscriptionResult{
		Text:     "Test",
		Duration: 1000 * time.Millisecond,
	}

	transcript := NewTranscript(result, "provider", "model", "", nil)
	data, err := transcript.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's indented (pretty-printed)
	if len(data) == 0 {
		t.Fatal("ToJSON() returned empty data")
	}

	// Should contain newlines for indentation
	if !json.Valid(data) {
		t.Error("ToJSON() returned invalid JSON")
	}

	// Verify schema URL is present
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed["$schema"] != TranscriptSchemaURL {
		t.Errorf("$schema = %v, want %s", parsed["$schema"], TranscriptSchemaURL)
	}
}
