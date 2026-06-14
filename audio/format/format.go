// Package format defines audio format types for voice processing.
package format

// AudioFormat describes an audio format for conversion.
type AudioFormat struct {
	// Encoding is the audio encoding ("pcm16", "mulaw", "alaw").
	Encoding string

	// SampleRate is the sample rate in Hz (8000, 16000, 24000).
	SampleRate int

	// Channels is the number of audio channels (1 = mono, 2 = stereo).
	Channels int
}

// Common audio formats used in voice gateways.
var (
	// Twilio is Twilio's native format (mulaw 8kHz mono).
	Twilio = AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}

	// Telnyx is Telnyx's native format (mulaw 8kHz mono).
	Telnyx = AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}

	// Vonage is Vonage's native format (PCM16 16kHz mono).
	Vonage = AudioFormat{Encoding: "pcm16", SampleRate: 16000, Channels: 1}

	// OpenAI is OpenAI Realtime's format (PCM16 24kHz mono).
	OpenAI = AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}

	// GeminiInput is Gemini Live's input format (PCM16 16kHz mono).
	GeminiInput = AudioFormat{Encoding: "pcm16", SampleRate: 16000, Channels: 1}

	// GeminiOutput is Gemini Live's output format (PCM16 24kHz mono).
	GeminiOutput = AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}
)

// Encoding-based aliases for provider-agnostic usage.
var (
	// PCM16_24kHz is PCM16 24kHz mono.
	PCM16_24kHz = OpenAI

	// PCM16_16kHz is PCM16 16kHz mono.
	PCM16_16kHz = GeminiInput
)
