// Package subtitle generates SRT and WebVTT subtitles from STT transcription results.
package subtitle

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/plexusone/omnivoice-core/stt"
)

// Options configures subtitle generation.
type Options struct {
	// MaxCharsPerLine limits characters per subtitle line.
	// Default: 42 (standard for readability).
	MaxCharsPerLine int

	// MaxLinesPerCue limits lines per subtitle cue.
	// Default: 2.
	MaxLinesPerCue int

	// MaxDuration limits the maximum duration for a single cue.
	// Default: 7 seconds.
	MaxDuration time.Duration

	// MinDuration sets the minimum duration for a cue to be readable.
	// Default: 1 second.
	MinDuration time.Duration

	// IncludeSpeakerLabels adds speaker labels to cues.
	IncludeSpeakerLabels bool
}

// DefaultOptions returns sensible defaults for subtitle generation.
func DefaultOptions() Options {
	return Options{
		MaxCharsPerLine:      42,
		MaxLinesPerCue:       2,
		MaxDuration:          7 * time.Second,
		MinDuration:          1 * time.Second,
		IncludeSpeakerLabels: false,
	}
}

// Format represents the output format for subtitles.
type Format int

const (
	// FormatSRT is the SubRip subtitle format.
	FormatSRT Format = iota
	// FormatVTT is the WebVTT subtitle format.
	FormatVTT
)

// GenerateSRT generates SRT subtitle content from a transcription result.
func GenerateSRT(result *stt.TranscriptionResult, opts Options) string {
	return generate(result, opts, FormatSRT)
}

// GenerateVTT generates WebVTT subtitle content from a transcription result.
func GenerateVTT(result *stt.TranscriptionResult, opts Options) string {
	return generate(result, opts, FormatVTT)
}

// SaveSRT generates and saves SRT content to a file.
func SaveSRT(result *stt.TranscriptionResult, filePath string, opts Options) error {
	content := GenerateSRT(result, opts)
	return os.WriteFile(filePath, []byte(content), 0600)
}

// SaveVTT generates and saves WebVTT content to a file.
func SaveVTT(result *stt.TranscriptionResult, filePath string, opts Options) error {
	content := GenerateVTT(result, opts)
	return os.WriteFile(filePath, []byte(content), 0600)
}

// generate is the common implementation for subtitle generation.
func generate(result *stt.TranscriptionResult, opts Options, format Format) string {
	if opts.MaxCharsPerLine == 0 {
		opts = DefaultOptions()
	}

	var sb strings.Builder

	// VTT header
	if format == FormatVTT {
		sb.WriteString("WEBVTT\n\n")
	}

	cueNum := 1

	for _, seg := range result.Segments {
		// Use words if available for better timing
		if len(seg.Words) > 0 {
			cues := wordsToSubtitleCues(seg.Words, opts)
			for _, cue := range cues {
				writeCue(&sb, cueNum, cue, opts, format)
				cueNum++
			}
		} else {
			// Fall back to segment-level timing
			cue := subtitleCue{
				StartTime: seg.StartTime,
				EndTime:   seg.EndTime,
				Text:      wrapText(seg.Text, opts.MaxCharsPerLine, opts.MaxLinesPerCue),
				Speaker:   seg.Speaker,
			}
			writeCue(&sb, cueNum, cue, opts, format)
			cueNum++
		}
	}

	return sb.String()
}

// writeCue writes a single subtitle cue to the builder.
func writeCue(sb *strings.Builder, cueNum int, cue subtitleCue, opts Options, format Format) {
	sb.WriteString(fmt.Sprintf("%d\n", cueNum))

	if format == FormatSRT {
		sb.WriteString(fmt.Sprintf("%s --> %s\n",
			formatSRTTime(cue.StartTime),
			formatSRTTime(cue.EndTime)))
		if opts.IncludeSpeakerLabels && cue.Speaker != "" {
			sb.WriteString(fmt.Sprintf("[%s] ", cue.Speaker))
		}
	} else {
		sb.WriteString(fmt.Sprintf("%s --> %s\n",
			formatVTTTime(cue.StartTime),
			formatVTTTime(cue.EndTime)))
		if opts.IncludeSpeakerLabels && cue.Speaker != "" {
			sb.WriteString(fmt.Sprintf("<v %s>", cue.Speaker))
		}
	}

	sb.WriteString(cue.Text)
	sb.WriteString("\n\n")
}

// subtitleCue represents a single subtitle cue.
type subtitleCue struct {
	StartTime time.Duration
	EndTime   time.Duration
	Text      string
	Speaker   string
}

// wordsToSubtitleCues groups words into subtitle cues based on options.
func wordsToSubtitleCues(words []stt.Word, opts Options) []subtitleCue {
	if len(words) == 0 {
		return nil
	}

	var cues []subtitleCue
	var currentWords []stt.Word
	var currentText strings.Builder
	var currentSpeaker string

	flushCue := func() {
		if len(currentWords) == 0 {
			return
		}
		cues = append(cues, subtitleCue{
			StartTime: currentWords[0].StartTime,
			EndTime:   currentWords[len(currentWords)-1].EndTime,
			Text:      wrapText(strings.TrimSpace(currentText.String()), opts.MaxCharsPerLine, opts.MaxLinesPerCue),
			Speaker:   currentSpeaker,
		})
		currentWords = nil
		currentText.Reset()
	}

	for _, word := range words {
		// Check if we need to start a new cue
		needNewCue := false

		// Speaker change
		if opts.IncludeSpeakerLabels && word.Speaker != currentSpeaker && len(currentWords) > 0 {
			needNewCue = true
		}

		// Text too long - check actual wrapped line count, not just total characters
		potentialText := currentText.String() + " " + word.Text
		wrappedText := wrapText(strings.TrimSpace(potentialText), opts.MaxCharsPerLine, opts.MaxLinesPerCue)
		lineCount := strings.Count(wrappedText, "\n") + 1
		if lineCount > opts.MaxLinesPerCue {
			needNewCue = true
		}

		// Duration too long
		if len(currentWords) > 0 {
			duration := word.EndTime - currentWords[0].StartTime
			if duration > opts.MaxDuration {
				needNewCue = true
			}
		}

		if needNewCue {
			flushCue()
		}

		// Add word to current cue
		if currentText.Len() > 0 {
			currentText.WriteString(" ")
		}
		currentText.WriteString(word.Text)
		currentWords = append(currentWords, word)
		currentSpeaker = word.Speaker
	}

	// Flush remaining words
	flushCue()

	return cues
}

// formatSRTTime formats a duration as SRT timestamp (HH:MM:SS,mmm).
func formatSRTTime(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	ms := d / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// formatVTTTime formats a duration as VTT timestamp (HH:MM:SS.mmm).
func formatVTTTime(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	d -= s * time.Second
	ms := d / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// wrapText wraps text to fit within character limits per line.
// Note: maxLines is advisory - we never drop words. If text exceeds maxLines,
// we continue wrapping rather than clipping, as dropped words appear as quality
// problems to viewers.
func wrapText(text string, maxCharsPerLine, _ int) string {
	if maxCharsPerLine <= 0 {
		maxCharsPerLine = 42
	}
	// maxLines is kept for API compatibility but not used for clipping

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= maxCharsPerLine {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}
