package stt

import (
	"encoding/json"
	"os"
	"time"

	"github.com/grokify/mogo/time/duration"
)

// TranscriptFormatVersion is the current version of the OmniVoice transcript format.
const TranscriptFormatVersion = "1.0"

// TranscriptSchemaURL is the JSON Schema URL for the transcript format.
const TranscriptSchemaURL = "https://omnivoice.dev/schema/transcript-v1.json"

// Transcript represents the OmniVoice JSON Transcript format.
// This is the canonical output format for transcription results.
type Transcript struct {
	// Schema is the JSON Schema URL for validation.
	Schema string `json:"$schema"`

	// Version is the format version (e.g., "1.0").
	Version string `json:"version"`

	// Text is the complete transcription text.
	Text string `json:"text"`

	// Language is the detected or specified language (BCP-47 code).
	Language string `json:"language,omitempty"`

	// LanguageConfidence is the confidence score for language detection (0.0-1.0).
	LanguageConfidence float64 `json:"language_confidence,omitempty"`

	// Duration is the audio duration (marshals as milliseconds in JSON).
	Duration duration.DurationMilliseconds `json:"duration_ms"`

	// Segments contains the transcript broken into segments.
	Segments []TranscriptSegment `json:"segments,omitempty"`

	// Metadata contains information about how the transcript was generated.
	Metadata TranscriptMetadata `json:"metadata"`
}

// TranscriptSegment represents a segment of the transcript (sentence, phrase, or utterance).
type TranscriptSegment struct {
	// Text is the transcribed text for this segment.
	Text string `json:"text"`

	// Start is the start time (marshals as milliseconds in JSON).
	Start duration.DurationMilliseconds `json:"start_ms"`

	// End is the end time (marshals as milliseconds in JSON).
	End duration.DurationMilliseconds `json:"end_ms"`

	// Speaker is the speaker identifier (if diarization is enabled).
	Speaker string `json:"speaker,omitempty"`

	// Confidence is the average confidence score for this segment (0.0-1.0).
	Confidence float64 `json:"confidence,omitempty"`

	// Language is the detected language for this segment (if different from overall).
	Language string `json:"language,omitempty"`

	// Words contains word-level details (if word timestamps are enabled).
	Words []TranscriptWord `json:"words,omitempty"`
}

// TranscriptWord represents a single word with timing information.
type TranscriptWord struct {
	// Text is the transcribed word.
	Text string `json:"text"`

	// Start is the start time (marshals as milliseconds in JSON).
	Start duration.DurationMilliseconds `json:"start_ms"`

	// End is the end time (marshals as milliseconds in JSON).
	End duration.DurationMilliseconds `json:"end_ms"`

	// Speaker is the speaker identifier (if diarization is enabled).
	Speaker string `json:"speaker,omitempty"`

	// Confidence is the recognition confidence (0.0-1.0).
	Confidence float64 `json:"confidence,omitempty"`
}

// TranscriptMetadata contains provenance information about the transcript.
type TranscriptMetadata struct {
	// Provider is the STT provider used (e.g., "deepgram", "openai").
	Provider string `json:"provider"`

	// Model is the provider-specific model used (if specified).
	Model string `json:"model,omitempty"`

	// CreatedAt is the ISO 8601 timestamp when the transcript was created.
	CreatedAt string `json:"created_at"`

	// AudioFile is the original audio file path or URL (if available).
	AudioFile string `json:"audio_file,omitempty"`

	// Options contains the transcription options that were used.
	Options *TranscriptOptions `json:"options,omitempty"`
}

// TranscriptOptions records the options used for transcription.
type TranscriptOptions struct {
	// Language is the requested language (if specified).
	Language string `json:"language,omitempty"`

	// EnablePunctuation indicates if punctuation was enabled.
	EnablePunctuation bool `json:"enable_punctuation,omitempty"`

	// EnableWordTimestamps indicates if word timestamps were enabled.
	EnableWordTimestamps bool `json:"enable_word_timestamps,omitempty"`

	// EnableSpeakerDiarization indicates if speaker diarization was enabled.
	EnableSpeakerDiarization bool `json:"enable_speaker_diarization,omitempty"`
}

// NewTranscript creates a Transcript from a TranscriptionResult.
func NewTranscript(result *TranscriptionResult, provider, model, audioFile string, config *TranscriptionConfig) *Transcript {
	t := &Transcript{
		Schema:             TranscriptSchemaURL,
		Version:            TranscriptFormatVersion,
		Text:               result.Text,
		Language:           result.Language,
		LanguageConfidence: result.LanguageConfidence,
		Duration:           duration.FromDuration(result.Duration),
		Segments:           make([]TranscriptSegment, len(result.Segments)),
		Metadata: TranscriptMetadata{
			Provider:  provider,
			Model:     model,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			AudioFile: audioFile,
		},
	}

	// Convert segments
	for i, seg := range result.Segments {
		t.Segments[i] = TranscriptSegment{
			Text:       seg.Text,
			Start:      duration.FromDuration(seg.StartTime),
			End:        duration.FromDuration(seg.EndTime),
			Speaker:    seg.Speaker,
			Confidence: seg.Confidence,
			Language:   seg.Language,
			Words:      make([]TranscriptWord, len(seg.Words)),
		}
		for j, word := range seg.Words {
			t.Segments[i].Words[j] = TranscriptWord{
				Text:       word.Text,
				Start:      duration.FromDuration(word.StartTime),
				End:        duration.FromDuration(word.EndTime),
				Speaker:    word.Speaker,
				Confidence: word.Confidence,
			}
		}
	}

	// Record options if provided
	if config != nil {
		t.Metadata.Options = &TranscriptOptions{
			Language:                 config.Language,
			EnablePunctuation:        config.EnablePunctuation,
			EnableWordTimestamps:     config.EnableWordTimestamps,
			EnableSpeakerDiarization: config.EnableSpeakerDiarization,
		}
	}

	return t
}

// ToJSON serializes the transcript to JSON with indentation.
func (t *Transcript) ToJSON() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// SaveJSON writes the transcript to a JSON file.
func (t *Transcript) SaveJSON(filePath string) error {
	data, err := t.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

// LoadTranscript reads a transcript from a JSON file.
func LoadTranscript(filePath string) (*Transcript, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var t Transcript
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// TotalDuration returns the total duration as a time.Duration.
func (t *Transcript) TotalDuration() time.Duration {
	return t.Duration.Duration()
}

// SegmentDuration returns the duration of a segment as time.Duration.
func (s *TranscriptSegment) SegmentDuration() time.Duration {
	return s.End.Duration() - s.Start.Duration()
}

// WordDuration returns the duration of a word as time.Duration.
func (w *TranscriptWord) WordDuration() time.Duration {
	return w.End.Duration() - w.Start.Duration()
}
