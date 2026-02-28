package subtitle

import (
	"strings"
	"testing"
	"time"

	"github.com/plexusone/omnivoice/stt"
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

func TestWordsToSubtitleCues_LineCountLimit(t *testing.T) {
	// This test verifies that cues are split based on actual wrapped line count,
	// not just total character count. The text "openai published a study revealing
	// that while everyone has access to ai only a small" is ~84 chars (within the
	// 42*2=84 char limit) but wraps to 3 lines, so "small" should start a new cue.
	words := []stt.Word{
		{Text: "openai", StartTime: 0, EndTime: 500 * time.Millisecond},
		{Text: "published", StartTime: 500 * time.Millisecond, EndTime: 900 * time.Millisecond},
		{Text: "a", StartTime: 900 * time.Millisecond, EndTime: 950 * time.Millisecond},
		{Text: "study", StartTime: 950 * time.Millisecond, EndTime: 1200 * time.Millisecond},
		{Text: "revealing", StartTime: 1200 * time.Millisecond, EndTime: 1600 * time.Millisecond},
		{Text: "that", StartTime: 1600 * time.Millisecond, EndTime: 1800 * time.Millisecond},
		{Text: "while", StartTime: 1800 * time.Millisecond, EndTime: 2000 * time.Millisecond},
		{Text: "everyone", StartTime: 2000 * time.Millisecond, EndTime: 2400 * time.Millisecond},
		{Text: "has", StartTime: 2400 * time.Millisecond, EndTime: 2600 * time.Millisecond},
		{Text: "access", StartTime: 2600 * time.Millisecond, EndTime: 2900 * time.Millisecond},
		{Text: "to", StartTime: 2900 * time.Millisecond, EndTime: 3000 * time.Millisecond},
		{Text: "ai", StartTime: 3000 * time.Millisecond, EndTime: 3200 * time.Millisecond},
		{Text: "only", StartTime: 3200 * time.Millisecond, EndTime: 3500 * time.Millisecond},
		{Text: "a", StartTime: 3500 * time.Millisecond, EndTime: 3600 * time.Millisecond},
		{Text: "small", StartTime: 3600 * time.Millisecond, EndTime: 3900 * time.Millisecond},
		{Text: "percentage", StartTime: 3900 * time.Millisecond, EndTime: 4500 * time.Millisecond},
	}

	opts := DefaultOptions() // MaxCharsPerLine=42, MaxLinesPerCue=2

	cues := wordsToSubtitleCues(words, opts)

	// Verify no cue has more than 2 lines
	for i, cue := range cues {
		lineCount := strings.Count(cue.Text, "\n") + 1
		if lineCount > opts.MaxLinesPerCue {
			t.Errorf("Cue %d has %d lines (max %d): %q", i, lineCount, opts.MaxLinesPerCue, cue.Text)
		}
	}

	// Verify "small" is in a cue (not dropped)
	foundSmall := false
	for _, cue := range cues {
		if strings.Contains(cue.Text, "small") {
			foundSmall = true
			break
		}
	}
	if !foundSmall {
		t.Error("Word 'small' should be present in output cues")
	}

	// The first cue should end with "a" (before "small")
	if !strings.HasSuffix(strings.TrimSpace(cues[0].Text), "a") {
		t.Errorf("First cue should end with 'a', got: %q", cues[0].Text)
	}

	// The second cue should start with "small"
	if !strings.HasPrefix(cues[1].Text, "small") {
		t.Errorf("Second cue should start with 'small', got: %q", cues[1].Text)
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

func TestWordsToSubtitleCues_EnglishWordGrouping(t *testing.T) {
	// Tests basic English word grouping: words are space-separated and all
	// words should be preserved in output cues.
	words := []stt.Word{
		{Text: "The", StartTime: 0, EndTime: 200 * time.Millisecond},
		{Text: "quick", StartTime: 200 * time.Millisecond, EndTime: 500 * time.Millisecond},
		{Text: "brown", StartTime: 500 * time.Millisecond, EndTime: 800 * time.Millisecond},
		{Text: "fox", StartTime: 800 * time.Millisecond, EndTime: 1 * time.Second},
		{Text: "jumps", StartTime: 1 * time.Second, EndTime: 1300 * time.Millisecond},
		{Text: "over", StartTime: 1300 * time.Millisecond, EndTime: 1600 * time.Millisecond},
		{Text: "the", StartTime: 1600 * time.Millisecond, EndTime: 1800 * time.Millisecond},
		{Text: "lazy", StartTime: 1800 * time.Millisecond, EndTime: 2100 * time.Millisecond},
		{Text: "dog.", StartTime: 2100 * time.Millisecond, EndTime: 2500 * time.Millisecond},
	}

	opts := DefaultOptions()
	cues := wordsToSubtitleCues(words, opts)

	// All words should fit in a single cue (sentence is short)
	if len(cues) != 1 {
		t.Errorf("Expected 1 cue for short sentence, got %d", len(cues))
	}

	// Verify all words are present and space-separated
	expectedWords := []string{"The", "quick", "brown", "fox", "jumps", "over", "the", "lazy", "dog."}
	for _, word := range expectedWords {
		if !strings.Contains(cues[0].Text, word) {
			t.Errorf("Expected word %q to be in cue, got: %q", word, cues[0].Text)
		}
	}

	// Verify timing spans entire input
	if cues[0].StartTime != 0 {
		t.Errorf("Expected start time 0, got %v", cues[0].StartTime)
	}
	if cues[0].EndTime != 2500*time.Millisecond {
		t.Errorf("Expected end time 2500ms, got %v", cues[0].EndTime)
	}
}

func TestWordsToSubtitleCues_ChineseCharacters(t *testing.T) {
	// Tests behavior when STT returns Chinese text character-by-character
	// (as Deepgram and some other services do). Each character is treated as
	// a separate "word", so they get space-separated in output. External
	// post-processing is needed to remove spaces between CJK characters.
	//
	// Input: "人工智能" (artificial intelligence) tokenized character-by-character
	words := []stt.Word{
		{Text: "人", StartTime: 0, EndTime: 200 * time.Millisecond},
		{Text: "工", StartTime: 200 * time.Millisecond, EndTime: 400 * time.Millisecond},
		{Text: "智", StartTime: 400 * time.Millisecond, EndTime: 600 * time.Millisecond},
		{Text: "能", StartTime: 600 * time.Millisecond, EndTime: 800 * time.Millisecond},
	}

	opts := DefaultOptions()
	cues := wordsToSubtitleCues(words, opts)

	if len(cues) != 1 {
		t.Errorf("Expected 1 cue, got %d", len(cues))
	}

	// Characters should be space-separated (current behavior)
	// External post-processing removes spaces for CJK output
	expected := "人 工 智 能"
	if cues[0].Text != expected {
		t.Errorf("Expected %q, got %q", expected, cues[0].Text)
	}

	// All characters should be preserved
	for _, char := range []string{"人", "工", "智", "能"} {
		if !strings.Contains(cues[0].Text, char) {
			t.Errorf("Expected character %q to be in cue", char)
		}
	}
}

func TestWordsToSubtitleCues_MixedChineseEnglish(t *testing.T) {
	// Tests mixed content where some words are English and some are
	// Chinese characters. This is common in technical content.
	//
	// Example: "AI 技术" (AI technology)
	words := []stt.Word{
		{Text: "AI", StartTime: 0, EndTime: 300 * time.Millisecond},
		{Text: "技", StartTime: 300 * time.Millisecond, EndTime: 500 * time.Millisecond},
		{Text: "术", StartTime: 500 * time.Millisecond, EndTime: 700 * time.Millisecond},
		{Text: "is", StartTime: 700 * time.Millisecond, EndTime: 900 * time.Millisecond},
		{Text: "important.", StartTime: 900 * time.Millisecond, EndTime: 1300 * time.Millisecond},
	}

	opts := DefaultOptions()
	cues := wordsToSubtitleCues(words, opts)

	if len(cues) != 1 {
		t.Errorf("Expected 1 cue, got %d", len(cues))
	}

	// All tokens should be present and space-separated
	expected := "AI 技 术 is important."
	if cues[0].Text != expected {
		t.Errorf("Expected %q, got %q", expected, cues[0].Text)
	}
}

func TestWordsToSubtitleCues_LongChineseText(t *testing.T) {
	// Tests that long Chinese text gets properly split into multiple cues
	// when it exceeds line limits. Each character counts as 1 char + 1 space.
	// With MaxCharsPerLine=42, ~20 characters fit per line.
	//
	// Input: 25 characters "这是一段很长的中文文本用于测试字幕分割功能是否正常工作"
	chars := []string{"这", "是", "一", "段", "很", "长", "的", "中", "文", "文",
		"本", "用", "于", "测", "试", "字", "幕", "分", "割", "功",
		"能", "是", "否", "正", "常", "工", "作"}

	words := make([]stt.Word, len(chars))
	for i, char := range chars {
		words[i] = stt.Word{
			Text:      char,
			StartTime: time.Duration(i*100) * time.Millisecond,
			EndTime:   time.Duration((i+1)*100) * time.Millisecond,
		}
	}

	opts := DefaultOptions()
	cues := wordsToSubtitleCues(words, opts)

	// Should produce multiple cues due to line limits
	if len(cues) < 2 {
		t.Errorf("Expected multiple cues for long text, got %d", len(cues))
	}

	// No cue should exceed 2 lines
	for i, cue := range cues {
		lineCount := strings.Count(cue.Text, "\n") + 1
		if lineCount > opts.MaxLinesPerCue {
			t.Errorf("Cue %d has %d lines (max %d): %q", i, lineCount, opts.MaxLinesPerCue, cue.Text)
		}
	}

	// All characters should be preserved across cues
	var allText strings.Builder
	for _, cue := range cues {
		allText.WriteString(cue.Text)
	}
	combined := allText.String()
	for _, char := range chars {
		if !strings.Contains(combined, char) {
			t.Errorf("Character %q missing from output", char)
		}
	}
}
