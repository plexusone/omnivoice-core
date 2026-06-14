// Package gateway provides a provider-agnostic interface for voice gateways.
//
// A Gateway handles full-duplex voice calls via telephony providers like Twilio
// or Telnyx. It manages HTTP webhooks, WebSocket media streams, and voice
// processing pipelines.
//
// # Pipeline Modes
//
// The gateway supports two pipeline modes:
//
// Text Pipeline (PipelineModeText):
//
//	Phone → Twilio WS → [STT → LLM (text) → TTS] → Twilio WS → Phone
//	Latency: ~500-1000ms (STT + LLM + TTS)
//
// Realtime Pipeline (PipelineModeRealtime):
//
//	Phone → Twilio WS → [realtime.Provider] → Twilio WS → Phone
//	Latency: ~100-200ms (native voice-to-voice)
//
// # Audio Format Conversion
//
// Telephony providers typically use mulaw 8kHz, while realtime providers use:
//   - OpenAI Realtime: PCM16 24kHz mono
//   - Gemini Live: PCM16 16kHz input, 24kHz output
//
// The gateway handles format conversion automatically.
//
// # Implementations
//
//   - github.com/plexusone/omni-twilio/omnivoice/gateway
//   - github.com/plexusone/omni-telnyx/omnivoice/gateway
//   - github.com/plexusone/omni-vonage/omnivoice/gateway
//   - github.com/plexusone/omni-plivo/omnivoice/gateway
//   - github.com/plexusone/omni-livekit/omnivoice/gateway
package gateway

import (
	"context"
	"log/slog"
	"time"

	"github.com/plexusone/omnivoice-core/realtime"
)

// ProviderName identifies a voice gateway provider.
type ProviderName string

const (
	// ProviderTwilio uses Twilio Media Streams.
	ProviderTwilio ProviderName = "twilio"

	// ProviderTelnyx uses Telnyx Media Streaming.
	ProviderTelnyx ProviderName = "telnyx"

	// ProviderVonage uses Vonage Voice WebSocket.
	ProviderVonage ProviderName = "vonage"

	// ProviderPlivo uses Plivo Audio Streaming.
	ProviderPlivo ProviderName = "plivo"

	// ProviderLiveKit uses LiveKit WebRTC.
	ProviderLiveKit ProviderName = "livekit"
)

// PipelineMode determines how audio is processed.
type PipelineMode string

const (
	// PipelineModeText uses the traditional STT → LLM → TTS pipeline.
	// Higher latency (~500-1000ms) but works with any LLM.
	PipelineModeText PipelineMode = "text"

	// PipelineModeRealtime uses native voice-to-voice via realtime.Provider.
	// Lower latency (~100-200ms) but requires OpenAI Realtime or Gemini Live.
	PipelineModeRealtime PipelineMode = "realtime"
)

// CallInfo contains information about a call.
type CallInfo struct {
	CallID    string
	From      string
	To        string
	Direction string
	StartTime time.Time
}

// CallHandler is called when a new call is received.
// Return nil to accept the call, or an error to reject it.
type CallHandler func(call *CallInfo) error

// Gateway defines the interface for voice gateway providers.
// Implementations handle provider-specific webhooks, media streams, and call control.
type Gateway interface {
	// Name returns the provider name.
	Name() ProviderName

	// Start starts the gateway server.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the gateway.
	Stop() error

	// OnCall sets the handler for incoming calls.
	OnCall(handler CallHandler)

	// MakeCall initiates an outbound call.
	MakeCall(ctx context.Context, to string) (Session, error)

	// GetSession retrieves an active session by call ID.
	GetSession(callID string) (Session, bool)

	// ListSessions returns all active sessions.
	ListSessions() []Session
}

// Session represents an active voice conversation session.
type Session interface {
	// ID returns the session identifier.
	ID() string

	// From returns the caller phone number.
	From() string

	// To returns the called phone number.
	To() string

	// Direction returns "inbound" or "outbound".
	Direction() string

	// StartTime returns when the session started.
	StartTime() time.Time

	// Duration returns the session duration.
	Duration() time.Duration

	// Events returns a channel for session events.
	Events() <-chan Event

	// Transcript returns the conversation transcript.
	Transcript() []Turn

	// Metrics returns session performance metrics.
	Metrics() Metrics

	// SendText sends text input to the agent (bypasses STT).
	SendText(text string) error

	// Interrupt stops the current agent speech.
	Interrupt()

	// Close ends the session.
	Close() error
}

// Turn represents a single conversation turn.
type Turn struct {
	Role       string     `json:"role"` // "user" or "agent"
	Text       string     `json:"text"`
	Timestamp  time.Time  `json:"timestamp"`
	DurationMs int        `json:"duration_ms"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation during conversation.
type ToolCall struct {
	Name       string         `json:"name"`
	Arguments  map[string]any `json:"arguments"`
	Result     string         `json:"result"`
	Error      string         `json:"error,omitempty"`
	DurationMs int            `json:"duration_ms"`
}

// Metrics contains session performance metrics.
type Metrics struct {
	SessionDurationMs     int   `json:"session_duration_ms"`
	TurnCount             int   `json:"turn_count"`
	UserSpeechDurationMs  int   `json:"user_speech_duration_ms"`
	AgentSpeechDurationMs int   `json:"agent_speech_duration_ms"`
	AvgSTTLatencyMs       int   `json:"avg_stt_latency_ms"`
	AvgLLMLatencyMs       int   `json:"avg_llm_latency_ms"`
	AvgTTSLatencyMs       int   `json:"avg_tts_latency_ms"`
	AvgTotalLatencyMs     int   `json:"avg_total_latency_ms"`
	InterruptionCount     int   `json:"interruption_count"`
	ToolCallCount         int   `json:"tool_call_count"`
	ErrorCount            int   `json:"error_count"`
	AudioBytesReceived    int64 `json:"audio_bytes_received"`
	AudioBytesSent        int64 `json:"audio_bytes_sent"`
}

// Event represents a session event.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
	Error     error     `json:"error,omitempty"`
}

// EventType identifies the type of session event.
type EventType string

const (
	EventSessionStarted   EventType = "session_started"
	EventSessionEnded     EventType = "session_ended"
	EventUserSpeechStart  EventType = "user_speech_start"
	EventUserSpeechEnd    EventType = "user_speech_end"
	EventUserTranscript   EventType = "user_transcript"
	EventAgentThinking    EventType = "agent_thinking"
	EventAgentSpeechStart EventType = "agent_speech_start"
	EventAgentSpeechEnd   EventType = "agent_speech_end"
	EventAgentTranscript  EventType = "agent_transcript"
	EventToolCall         EventType = "tool_call"
	EventInterruption     EventType = "interruption"
	EventError            EventType = "error"
	EventAudioReceived    EventType = "audio_received"
	EventAudioSent        EventType = "audio_sent"
)

// Config provides common configuration for voice gateways.
type Config struct {
	// Server configuration
	ListenAddr string // e.g., ":8080"
	PublicURL  string // e.g., "https://your-server.com"

	// Pipeline mode selection
	// Default: PipelineModeText if RealtimeProvider is nil
	Mode PipelineMode

	// Text pipeline configuration (used when Mode == PipelineModeText)
	STTProvider string
	STTAPIKey   string
	STTModel    string
	STTLanguage string

	TTSProvider string
	TTSAPIKey   string
	TTSVoiceID  string
	TTSModel    string

	LLMProvider     string
	LLMAPIKey       string
	LLMModel        string
	LLMSystemPrompt string

	// Realtime pipeline configuration (used when Mode == PipelineModeRealtime)
	// Provide either RealtimeProvider directly, or RealtimeConfig to create one.
	RealtimeProvider realtime.Provider
	RealtimeConfig   *RealtimeConfig

	// Session configuration
	MaxSessionDuration time.Duration
	InterruptionMode   string // "immediate", "after_sentence", "disabled"

	// Logging
	Logger *slog.Logger
}

// RealtimeConfig configures a realtime provider for voice-to-voice.
type RealtimeConfig struct {
	// Provider is the realtime provider name ("openai" or "gemini").
	Provider string `json:"provider"`

	// APIKey is the API key for the realtime provider.
	APIKey string `json:"api_key"`

	// Model is the model to use (e.g., "gpt-4o-realtime-preview-2024-12-17").
	Model string `json:"model,omitempty"`

	// Voice is the voice for audio output (e.g., "alloy", "Puck").
	Voice string `json:"voice,omitempty"`

	// Instructions is the system prompt for the conversation.
	Instructions string `json:"instructions,omitempty"`

	// Functions are tools the model can call during conversation.
	Functions []realtime.FunctionDeclaration `json:"functions,omitempty"`

	// OnFunctionCall handles function calls from the model.
	// If nil, function calls are ignored.
	OnFunctionCall func(id, name, args string) (result any, err error) `json:"-"`

	// Temperature controls response randomness (0-2).
	Temperature float64 `json:"temperature,omitempty"`
}

// ToProcessConfig converts RealtimeConfig to realtime.ProcessConfig.
func (c *RealtimeConfig) ToProcessConfig() realtime.ProcessConfig {
	return realtime.ProcessConfig{
		Instructions:   c.Instructions,
		Voice:          c.Voice,
		Functions:      c.Functions,
		OnFunctionCall: c.OnFunctionCall,
		Temperature:    c.Temperature,
	}
}

// LLMProvider defines the interface for LLM integration with voice gateways.
// Used in text pipeline mode (PipelineModeText).
type LLMProvider interface {
	// Generate produces a response given user input and conversation history.
	Generate(ctx context.Context, input string, history []Turn) (response string, toolCalls []ToolCall, err error)
}

// RealtimeProviderFactory creates realtime providers from configuration.
// Implementations should be registered for each supported provider.
type RealtimeProviderFactory interface {
	// Create creates a realtime provider from the given configuration.
	Create(config *RealtimeConfig) (realtime.Provider, error)

	// Name returns the provider name (e.g., "openai", "gemini").
	Name() string
}

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
	// AudioFormatTwilio is Twilio's native format (mulaw 8kHz mono).
	AudioFormatTwilio = AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}

	// AudioFormatTelnyx is Telnyx's native format (mulaw 8kHz mono).
	AudioFormatTelnyx = AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}

	// AudioFormatOpenAI is OpenAI Realtime's format (PCM16 24kHz mono).
	AudioFormatOpenAI = AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}

	// AudioFormatGeminiInput is Gemini Live's input format (PCM16 16kHz mono).
	AudioFormatGeminiInput = AudioFormat{Encoding: "pcm16", SampleRate: 16000, Channels: 1}

	// AudioFormatGeminiOutput is Gemini Live's output format (PCM16 24kHz mono).
	AudioFormatGeminiOutput = AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}
)

// AudioConverter converts audio between formats.
// Used to bridge telephony audio (mulaw 8kHz) with realtime providers (PCM16 16/24kHz).
type AudioConverter interface {
	// Convert converts audio from the source format to the target format.
	Convert(audio []byte, from, to AudioFormat) ([]byte, error)
}
