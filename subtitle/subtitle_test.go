package subtitle

import (
	"strings"
	"testing"
	"time"

	"github.com/agentplexus/omnivoice/stt"
)

func TestGenerateSRT(t *testing.T) {
	result := &stt.TranscriptionResult{
		Text: "Hello world. How are you today?",
		Segments: []stt.Segment{
			{
				Text:      "Hello world.",
				StartTime: 0,
				EndTime:   2 * time.Second,
				Words: []stt.Word{
					{Text: "Hello", StartTime: 0, EndTime: 500 * time.Millisecond},
					{Text: "world.", StartTime: 600 * time.Millisecond, EndTime: 2 * time.Second},
				},
			},
			{
				Text:      "How are you today?",
				StartTime: 2500 * time.Millisecond,
				EndTime:   5 * time.Second,
				Words: []stt.Word{
					{Text: "How", StartTime: 2500 * time.Millisecond, EndTime: 2800 * time.Millisecond},
					{Text: "are", StartTime: 2900 * time.Millisecond, EndTime: 3200 * time.Millisecond},
					{Text: "you", StartTime: 3300 * time.Millisecond, EndTime: 3600 * time.Millisecond},
					{Text: "today?", StartTime: 3700 * time.Millisecond, EndTime: 5 * time.Second},
				},
			},
		},
	}

	srtContent := GenerateSRT(result, DefaultOptions())

	// Check basic structure
	if !strings.Contains(srtContent, "1\n") {
		t.Error("SRT should contain cue number 1")
	}
	if !strings.Contains(srtContent, "-->") {
		t.Error("SRT should contain arrow separator")
	}
	if !strings.Contains(srtContent, ",") {
		t.Error("SRT timestamps should use comma separator")
	}
	if !strings.Contains(srtContent, "Hello") {
		t.Error("SRT should contain transcribed text")
	}
}

func TestGenerateVTT(t *testing.T) {
	result := &stt.TranscriptionResult{
		Text: "Hello world.",
		Segments: []stt.Segment{
			{
				Text:      "Hello world.",
				StartTime: 0,
				EndTime:   2 * time.Second,
			},
		},
	}

	vttContent := GenerateVTT(result, DefaultOptions())

	// Check VTT header
	if !strings.HasPrefix(vttContent, "WEBVTT") {
		t.Error("VTT should start with WEBVTT header")
	}
	// Check VTT uses period for milliseconds
	if !strings.Contains(vttContent, ".") {
		t.Error("VTT timestamps should use period separator")
	}
}

func TestFormatSRTTime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00:00,000"},
		{1500 * time.Millisecond, "00:00:01,500"},
		{time.Hour + 2*time.Minute + 3*time.Second + 456*time.Millisecond, "01:02:03,456"},
	}

	for _, tt := range tests {
		got := formatSRTTime(tt.duration)
		if got != tt.expected {
			t.Errorf("formatSRTTime(%v) = %q, want %q", tt.duration, got, tt.expected)
		}
	}
}

func TestFormatVTTTime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00:00.000"},
		{1500 * time.Millisecond, "00:00:01.500"},
		{time.Hour + 2*time.Minute + 3*time.Second + 456*time.Millisecond, "01:02:03.456"},
	}

	for _, tt := range tests {
		got := formatVTTTime(tt.duration)
		if got != tt.expected {
			t.Errorf("formatVTTTime(%v) = %q, want %q", tt.duration, got, tt.expected)
		}
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text            string
		maxCharsPerLine int
		maxLines        int
		expected        string
	}{
		{"Hello world", 42, 2, "Hello world"},
		// Note: wrapText no longer clips to maxLines - it wraps all words to avoid dropping content
		{"Hello world this is a test", 10, 2, "Hello\nworld this\nis a test"},
		{"Short", 42, 2, "Short"},
		{"", 42, 2, ""},
	}

	for _, tt := range tests {
		got := wrapText(tt.text, tt.maxCharsPerLine, tt.maxLines)
		if got != tt.expected {
			t.Errorf("wrapText(%q, %d, %d) = %q, want %q",
				tt.text, tt.maxCharsPerLine, tt.maxLines, got, tt.expected)
		}
	}
}

func TestSpeakerLabels(t *testing.T) {
	result := &stt.TranscriptionResult{
		Segments: []stt.Segment{
			{
				Text:      "Hello from speaker one.",
				StartTime: 0,
				EndTime:   2 * time.Second,
				Speaker:   "Speaker 1",
			},
		},
	}

	opts := DefaultOptions()
	opts.IncludeSpeakerLabels = true

	srtContent := GenerateSRT(result, opts)
	if !strings.Contains(srtContent, "[Speaker 1]") {
		t.Error("SRT with speaker labels should contain speaker tag")
	}

	vttContent := GenerateVTT(result, opts)
	if !strings.Contains(vttContent, "<v Speaker 1>") {
		t.Error("VTT with speaker labels should contain voice tag")
	}
}
