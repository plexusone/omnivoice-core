package realtime

import (
	"context"
	"encoding/json"
)

// Provider defines the interface for real-time voice-to-voice providers.
//
// Real-time providers handle bidirectional audio streaming, enabling native
// voice conversations with low latency (~100-300ms). Unlike traditional
// STT+LLM+TTS pipelines, real-time providers process audio directly.
type Provider interface {
	// ProcessAudioStream starts a real-time voice session.
	//
	// audioIn receives raw audio chunks from the user (microphone, telephony).
	// The audio format depends on the provider (typically PCM16 16-24kHz mono).
	//
	// Returns two channels:
	//   - audioCh: audio chunks from the model (PCM16 24kHz mono)
	//   - transcriptCh: transcript updates (both user input and model output)
	//
	// Both channels are closed when the session ends (context cancelled,
	// audioIn closed, or error).
	ProcessAudioStream(ctx context.Context, audioIn <-chan []byte, config ProcessConfig) (
		audioCh <-chan AudioChunk,
		transcriptCh <-chan Transcript,
		err error,
	)

	// Name returns the provider name (e.g., "openai-realtime", "gemini-live").
	Name() string

	// Close releases any resources held by the provider.
	Close() error
}

// ProcessConfig configures a real-time audio processing session.
type ProcessConfig struct {
	// Instructions is the system prompt for the conversation.
	Instructions string

	// Voice is the voice identifier for audio output.
	// Provider-specific (e.g., "alloy", "Puck").
	Voice string

	// Functions are functions the model can call during the conversation.
	Functions []FunctionDeclaration

	// OnFunctionCall is called when the model invokes a function.
	// The handler should execute the function and return the result.
	//
	// Parameters:
	//   - id: unique identifier for this function call
	//   - name: function name being called
	//   - args: JSON-encoded function arguments
	//
	// Returns:
	//   - result: any JSON-serializable value to return to the model
	//   - error: if non-nil, sent as error response to the model
	OnFunctionCall func(id, name, args string) (result any, err error)

	// Temperature controls response randomness (0.0 to 2.0).
	// Default varies by provider.
	Temperature float64

	// Extensions holds provider-specific settings.
	// Keys should be namespaced by provider (e.g., "openai.turn_detection").
	Extensions map[string]any
}

// FunctionDeclaration describes a function the model can call.
type FunctionDeclaration struct {
	// Name is the function name.
	Name string `json:"name"`

	// Description explains what the function does.
	Description string `json:"description"`

	// Parameters is a JSON Schema describing the function parameters.
	// Use json.RawMessage for flexibility across providers.
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// AudioChunk represents a chunk of audio data from the model.
type AudioChunk struct {
	// Audio is the raw audio data.
	// Format is PCM16 (signed 16-bit little-endian) at 24kHz mono.
	Audio []byte

	// IsFinal indicates this is the last chunk for the current turn.
	// Use this to know when the model has finished speaking.
	IsFinal bool
}

// Transcript represents a transcript update during the conversation.
type Transcript struct {
	// Text is the transcript text.
	Text string

	// IsFinal indicates this is a final (non-interim) transcript.
	// Interim transcripts may be revised; final transcripts are stable.
	IsFinal bool

	// IsInput indicates this is user input transcription.
	// If false, this is model output transcription.
	IsInput bool

	// ItemID is a provider-specific identifier for this transcript item.
	// Can be used to correlate with audio chunks.
	ItemID string
}

// Role returns the role associated with this transcript.
func (t Transcript) Role() string {
	if t.IsInput {
		return "user"
	}
	return "assistant"
}

// AudioFormat describes the audio encoding parameters.
type AudioFormat struct {
	// Encoding is the audio encoding ("pcm16", "mulaw", "alaw").
	Encoding string

	// SampleRate is the sample rate in Hz (e.g., 16000, 24000).
	SampleRate int

	// Channels is the number of audio channels (1 = mono, 2 = stereo).
	Channels int
}

// Common audio formats used by real-time providers.
var (
	// FormatPCM16_24kHz is PCM16 24kHz mono, used by OpenAI Realtime.
	FormatPCM16_24kHz = AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}

	// FormatPCM16_16kHz is PCM16 16kHz mono, used by Gemini Live input.
	FormatPCM16_16kHz = AudioFormat{Encoding: "pcm16", SampleRate: 16000, Channels: 1}
)
