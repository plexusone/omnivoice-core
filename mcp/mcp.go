// Package mcp provides an MCP (Model Context Protocol) server for voice interactions.
//
// The MCP server exposes voice interaction tools that can be used by Claude Code
// or other MCP-compatible clients. It abstracts the underlying STT, TTS, and
// transport providers, allowing any combination of providers to be used.
//
// # Tools
//
// The server exposes four tools:
//
//   - initiate_call: Start a voice call with an initial message
//   - continue_call: Continue an active call with a follow-up message
//   - speak_to_user: Speak a message without waiting for a response
//   - end_call: End an active call with a closing message
//
// # Usage
//
// Create a server with your providers and run it:
//
//	server := mcp.NewServer(mcp.Config{
//	    STT:       sttProvider,       // stt.StreamingProvider
//	    TTS:       ttsProvider,       // tts.StreamingProvider
//	    Transport: transportProvider, // transport.Transport
//	    PhoneNumber:     "+15551234567",
//	    UserPhoneNumber: "+15559876543",
//	})
//	server.Run(ctx) // Blocks, communicates via stdio
package mcp

import (
	"github.com/plexusone/omnivoice-core/stt"
	"github.com/plexusone/omnivoice-core/transport"
	"github.com/plexusone/omnivoice-core/tts"
)

// Config configures the MCP server.
type Config struct {
	// STT is the speech-to-text provider.
	STT stt.StreamingProvider

	// TTS is the text-to-speech provider.
	TTS tts.StreamingProvider

	// Transport is the audio transport provider.
	Transport transport.Transport

	// PhoneNumber is the phone number to call from (E.164 format).
	PhoneNumber string

	// UserPhoneNumber is the phone number to call (E.164 format).
	UserPhoneNumber string

	// TranscriptTimeoutMs is the timeout for waiting for user speech.
	// Default is 180000 (3 minutes).
	TranscriptTimeoutMs int

	// VoiceID is the TTS voice to use.
	VoiceID string

	// TTSModel is the TTS model to use.
	TTSModel string

	// STTModel is the STT model to use.
	STTModel string

	// STTLanguage is the STT language code (e.g., "en-US").
	STTLanguage string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		TranscriptTimeoutMs: 180000, // 3 minutes
		STTLanguage:         "en-US",
	}
}
