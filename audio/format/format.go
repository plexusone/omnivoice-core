// Package format defines audio format types for voice processing.
package format

import "strings"

// Encoding represents an audio encoding format name.
type Encoding string

// Standard encoding names used across voice providers.
const (
	// Linear16 is 16-bit linear PCM (also called PCM16, pcm_s16le).
	// This is a raw format that requires explicit sample rate and channels.
	Linear16 Encoding = "linear16"

	// MuLaw is G.711 µ-law encoding (telephony standard in North America/Japan).
	// This is a raw format that requires explicit sample rate and channels.
	MuLaw Encoding = "mulaw"

	// ALaw is G.711 A-law encoding (telephony standard in Europe).
	// This is a raw format that requires explicit sample rate and channels.
	ALaw Encoding = "alaw"

	// MP3 is MPEG Audio Layer III compressed format.
	// This is a container format with embedded metadata.
	MP3 Encoding = "mp3"

	// Opus is the Opus compressed audio format.
	// This is a container format with embedded metadata.
	Opus Encoding = "opus"

	// FLAC is Free Lossless Audio Codec.
	// This is a container format with embedded metadata.
	FLAC Encoding = "flac"

	// AAC is Advanced Audio Coding.
	// This is a container format with embedded metadata.
	AAC Encoding = "aac"

	// Speex is the Speex speech codec.
	// Optimized for voice, used in some VoIP applications.
	Speex Encoding = "speex"

	// WebM is the WebM container format (typically contains Opus or Vorbis).
	// This is a container format with embedded metadata.
	WebM Encoding = "webm"
)

// Normalize returns a canonical encoding name (lowercase, trimmed).
// This handles common variations:
//   - Case insensitivity: "LINEAR16" → "linear16"
//   - Whitespace: " mulaw " → "mulaw"
//   - Aliases: "pcm16" → "linear16", "ulaw" → "mulaw", "g711a" → "alaw"
//   - Container mappings: "wav" → "linear16" (WAV contains raw PCM)
func (e Encoding) Normalize() Encoding {
	s := strings.ToLower(strings.TrimSpace(string(e)))
	switch s {
	case "linear16", "pcm16", "pcm", "pcm_s16le", "wav":
		// WAV is a container for raw PCM, so normalize to linear16
		return Linear16
	case "mulaw", "ulaw", "g711u", "pcm_mulaw":
		return MuLaw
	case "alaw", "g711a", "pcm_alaw":
		return ALaw
	case "mp3":
		return MP3
	case "opus":
		return Opus
	case "flac":
		return FLAC
	case "aac":
		return AAC
	case "speex":
		return Speex
	case "webm":
		return WebM
	default:
		return Encoding(s)
	}
}

// String returns the encoding as a string.
func (e Encoding) String() string {
	return string(e)
}

// IsRaw returns true if the encoding is a raw audio format without container headers.
// Raw formats (linear16, mulaw, alaw) require explicit sample rate and channels
// because the audio data contains no metadata.
// Container formats (mp3, opus, flac, aac, wav) include headers with metadata
// and can be auto-detected by audio processing services.
func (e Encoding) IsRaw() bool {
	switch e.Normalize() {
	case Linear16, MuLaw, ALaw:
		return true
	default:
		return false
	}
}

// IsRawEncoding returns true if the encoding string represents a raw audio format.
// This is a convenience function that normalizes the input and checks if it's raw.
// It handles case insensitivity, whitespace, and common aliases.
//
// Example:
//
//	IsRawEncoding("linear16")  // true
//	IsRawEncoding("PCM16")     // true (normalized to linear16)
//	IsRawEncoding(" mulaw ")   // true (trimmed)
//	IsRawEncoding("mp3")       // false (container format)
func IsRawEncoding(encoding string) bool {
	return Encoding(encoding).IsRaw()
}

// AudioFormat describes an audio format for conversion.
type AudioFormat struct {
	// Encoding is the audio encoding (e.g., Linear16, MuLaw, ALaw).
	Encoding string

	// SampleRate is the sample rate in Hz (8000, 16000, 24000).
	SampleRate int

	// Channels is the number of audio channels (1 = mono, 2 = stereo).
	Channels int
}

// Common audio formats used in voice gateways.
var (
	// Twilio is Twilio's native format (mulaw 8kHz mono).
	Twilio = AudioFormat{Encoding: string(MuLaw), SampleRate: 8000, Channels: 1}

	// Telnyx is Telnyx's native format (mulaw 8kHz mono).
	Telnyx = AudioFormat{Encoding: string(MuLaw), SampleRate: 8000, Channels: 1}

	// Vonage is Vonage's native format (PCM16 16kHz mono).
	Vonage = AudioFormat{Encoding: string(Linear16), SampleRate: 16000, Channels: 1}

	// OpenAI is OpenAI Realtime's format (PCM16 24kHz mono).
	OpenAI = AudioFormat{Encoding: string(Linear16), SampleRate: 24000, Channels: 1}

	// GeminiInput is Gemini Live's input format (PCM16 16kHz mono).
	GeminiInput = AudioFormat{Encoding: string(Linear16), SampleRate: 16000, Channels: 1}

	// GeminiOutput is Gemini Live's output format (PCM16 24kHz mono).
	GeminiOutput = AudioFormat{Encoding: string(Linear16), SampleRate: 24000, Channels: 1}
)

// Encoding-based aliases for provider-agnostic usage.
var (
	// PCM16_24kHz is PCM16 24kHz mono.
	PCM16_24kHz = OpenAI

	// PCM16_16kHz is PCM16 16kHz mono.
	PCM16_16kHz = GeminiInput
)
