# Transcript Format

OmniVoice provides a canonical JSON transcript format for STT (speech-to-text) output. This standardized format enables consistent handling of transcription results across different providers and applications.

## Overview

The Transcript format captures:

- Full transcription text
- Segment-level timing (sentences, phrases, utterances)
- Word-level timing (when enabled)
- Speaker diarization (when enabled)
- Confidence scores
- Provider metadata and options used

All duration fields use `DurationMilliseconds` which serializes as integer milliseconds in JSON while providing full `time.Duration` functionality in Go.

## Quick Start

```go
import "github.com/plexusone/omnivoice-core/stt"

// Convert a transcription result to canonical format
transcript := stt.NewTranscript(result, "deepgram", "nova-2", "audio.mp3", config)

// Save to JSON file
err := transcript.SaveJSON("output.transcript.json")

// Load from JSON file
loaded, err := stt.LoadTranscript("output.transcript.json")

// Access timing information
fmt.Printf("Total duration: %v\n", transcript.TotalDuration())
for _, seg := range transcript.Segments {
    fmt.Printf("[%v - %v] %s\n",
        seg.Start.Duration(),
        seg.End.Duration(),
        seg.Text)
}
```

## JSON Format

```json
{
  "$schema": "https://omnivoice.dev/schema/transcript-v1.json",
  "version": "1.0",
  "text": "Hello world. How are you today?",
  "language": "en-US",
  "language_confidence": 0.95,
  "duration_ms": 5000,
  "segments": [
    {
      "text": "Hello world.",
      "start_ms": 0,
      "end_ms": 1500,
      "speaker": "speaker_1",
      "confidence": 0.98,
      "words": [
        {
          "text": "Hello",
          "start_ms": 0,
          "end_ms": 600,
          "confidence": 0.99
        },
        {
          "text": "world.",
          "start_ms": 700,
          "end_ms": 1500,
          "confidence": 0.97
        }
      ]
    },
    {
      "text": "How are you today?",
      "start_ms": 2000,
      "end_ms": 4500,
      "speaker": "speaker_2",
      "confidence": 0.96
    }
  ],
  "metadata": {
    "provider": "deepgram",
    "model": "nova-2",
    "created_at": "2026-05-02T12:00:00Z",
    "audio_file": "conversation.mp3",
    "options": {
      "language": "en-US",
      "enable_punctuation": true,
      "enable_word_timestamps": true,
      "enable_speaker_diarization": true
    }
  }
}
```

## Types

### Transcript

The root type containing the full transcription.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| `Schema` | `string` | `$schema` | JSON Schema URL for validation |
| `Version` | `string` | `version` | Format version (currently "1.0") |
| `Text` | `string` | `text` | Complete transcription text |
| `Language` | `string` | `language` | BCP-47 language code (e.g., "en-US") |
| `LanguageConfidence` | `float64` | `language_confidence` | Language detection confidence (0.0-1.0) |
| `Duration` | `DurationMilliseconds` | `duration_ms` | Total audio duration |
| `Segments` | `[]TranscriptSegment` | `segments` | Transcript segments |
| `Metadata` | `TranscriptMetadata` | `metadata` | Provenance information |

### TranscriptSegment

A segment of the transcript (sentence, phrase, or utterance).

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| `Text` | `string` | `text` | Segment text |
| `Start` | `DurationMilliseconds` | `start_ms` | Start time |
| `End` | `DurationMilliseconds` | `end_ms` | End time |
| `Speaker` | `string` | `speaker` | Speaker identifier |
| `Confidence` | `float64` | `confidence` | Average confidence (0.0-1.0) |
| `Language` | `string` | `language` | Segment language (if different) |
| `Words` | `[]TranscriptWord` | `words` | Word-level details |

### TranscriptWord

A single word with timing information.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| `Text` | `string` | `text` | The word |
| `Start` | `DurationMilliseconds` | `start_ms` | Start time |
| `End` | `DurationMilliseconds` | `end_ms` | End time |
| `Speaker` | `string` | `speaker` | Speaker identifier |
| `Confidence` | `float64` | `confidence` | Recognition confidence (0.0-1.0) |

### TranscriptMetadata

Provenance information about how the transcript was generated.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| `Provider` | `string` | `provider` | STT provider (e.g., "deepgram", "openai") |
| `Model` | `string` | `model` | Provider-specific model |
| `CreatedAt` | `string` | `created_at` | ISO 8601 timestamp |
| `AudioFile` | `string` | `audio_file` | Original audio file path/URL |
| `Options` | `*TranscriptOptions` | `options` | Transcription options used |

### TranscriptOptions

Records the options used for transcription.

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| `Language` | `string` | `language` | Requested language |
| `EnablePunctuation` | `bool` | `enable_punctuation` | Punctuation enabled |
| `EnableWordTimestamps` | `bool` | `enable_word_timestamps` | Word timestamps enabled |
| `EnableSpeakerDiarization` | `bool` | `enable_speaker_diarization` | Speaker diarization enabled |

## DurationMilliseconds

Duration fields use `duration.DurationMilliseconds` from `github.com/grokify/mogo/time/duration`. This type:

- Wraps `time.Duration` for full Go duration functionality
- Serializes as integer milliseconds in JSON (not nanoseconds)
- Provides type safety to prevent mixing with raw integers

```go
import "github.com/grokify/mogo/time/duration"

// Create from time.Duration
d := duration.FromDuration(5 * time.Second)

// Create from milliseconds
d := duration.FromMilliseconds(5000)

// Access as time.Duration
td := d.Duration()
td.Seconds() // 5.0

// Get milliseconds
d.Milliseconds() // 5000

// JSON marshaling
data, _ := json.Marshal(d) // "5000"
```

## Methods

### Transcript Methods

```go
// TotalDuration returns the total duration as time.Duration
func (t *Transcript) TotalDuration() time.Duration

// ToJSON serializes the transcript to indented JSON
func (t *Transcript) ToJSON() ([]byte, error)

// SaveJSON writes the transcript to a JSON file
func (t *Transcript) SaveJSON(filePath string) error
```

### TranscriptSegment Methods

```go
// SegmentDuration returns the segment duration as time.Duration
func (s *TranscriptSegment) SegmentDuration() time.Duration
```

### TranscriptWord Methods

```go
// WordDuration returns the word duration as time.Duration
func (w *TranscriptWord) WordDuration() time.Duration
```

## Functions

```go
// NewTranscript creates a Transcript from a TranscriptionResult
func NewTranscript(
    result *TranscriptionResult,
    provider, model, audioFile string,
    config *TranscriptionConfig,
) *Transcript

// LoadTranscript reads a transcript from a JSON file
func LoadTranscript(filePath string) (*Transcript, error)
```

## Schema Validation

The `schema` package provides an embedded JSON Schema for validation:

```go
import "github.com/plexusone/omnivoice-core/schema"

// Get the embedded schema
schemaJSON := schema.TranscriptV1Schema

// Use with any JSON Schema validator library
// Example with github.com/santhosh-tekuri/jsonschema:
compiler := jsonschema.NewCompiler()
if err := compiler.AddResource("transcript.json", strings.NewReader(schemaJSON)); err != nil {
    log.Fatal(err)
}
sch, err := compiler.Compile("transcript.json")
if err != nil {
    log.Fatal(err)
}

// Validate a transcript
transcriptData, _ := transcript.ToJSON()
if err := sch.Validate(bytes.NewReader(transcriptData)); err != nil {
    log.Printf("Validation failed: %v", err)
}
```

## Constants

```go
// TranscriptFormatVersion is the current version of the format
const TranscriptFormatVersion = "1.0"

// TranscriptSchemaURL is the JSON Schema URL
const TranscriptSchemaURL = "https://omnivoice.dev/schema/transcript-v1.json"
```

## Use Cases

### Converting Provider Results

```go
// After transcribing with any STT provider
result, err := provider.Transcribe(ctx, audioData, config)
if err != nil {
    return err
}

// Convert to canonical format
transcript := stt.NewTranscript(result, provider.Name(), "whisper-1", "recording.wav", config)

// Save for later analysis
return transcript.SaveJSON("transcripts/recording.transcript.json")
```

### Building Subtitles

```go
import "github.com/plexusone/omnivoice-core/subtitle"

// Load existing transcript
transcript, err := stt.LoadTranscript("recording.transcript.json")
if err != nil {
    return err
}

// Generate SRT subtitles from segments
// (subtitle package works with TranscriptionResult,
// but you can convert Transcript segments back)
```

### Analyzing Speaker Turns

```go
transcript, _ := stt.LoadTranscript("meeting.transcript.json")

speakers := make(map[string]time.Duration)
for _, seg := range transcript.Segments {
    speakers[seg.Speaker] += seg.SegmentDuration()
}

for speaker, duration := range speakers {
    fmt.Printf("%s spoke for %v\n", speaker, duration)
}
```

### Cross-Application Interop

The canonical JSON format enables sharing transcripts between applications:

```go
// Application A: Generate transcript
transcript := stt.NewTranscript(result, "deepgram", "nova-2", "audio.mp3", config)
transcript.SaveJSON("shared/transcript.json")

// Application B: Load and process
loaded, _ := stt.LoadTranscript("shared/transcript.json")
fmt.Printf("Transcribed by %s using %s\n",
    loaded.Metadata.Provider,
    loaded.Metadata.Model)
```
