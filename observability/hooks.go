package observability

import (
	"context"
	"io"
	"time"
)

// TTSCallInfo contains information about a TTS operation.
type TTSCallInfo struct {
	// CallID is a unique identifier for correlating BeforeRequest/AfterResponse.
	CallID string

	// Provider is the TTS provider name (e.g., "elevenlabs", "openai").
	Provider string

	// StartTime is when the operation started.
	StartTime time.Time

	// VoiceID is the voice being used.
	VoiceID string

	// Model is the TTS model being used.
	Model string
}

// TTSRequest contains the TTS request details.
type TTSRequest struct {
	// Text is the text to synthesize.
	Text string

	// TextLength is the character count.
	TextLength int

	// OutputFormat is the audio format (e.g., "mp3", "pcm", "ulaw").
	OutputFormat string

	// SampleRate is the audio sample rate.
	SampleRate int
}

// TTSResponse contains the TTS response details.
type TTSResponse struct {
	// AudioSize is the size of the generated audio in bytes.
	AudioSize int64

	// Duration is the duration of the generated audio.
	Duration time.Duration

	// Latency is the time from request to first byte (for streaming).
	Latency time.Duration
}

// TTSHook provides observability for TTS operations.
type TTSHook interface {
	// BeforeSynthesize is called before TTS synthesis.
	// Returns a potentially modified context for propagation.
	BeforeSynthesize(ctx context.Context, info TTSCallInfo, req TTSRequest) context.Context

	// AfterSynthesize is called after TTS synthesis completes.
	AfterSynthesize(ctx context.Context, info TTSCallInfo, req TTSRequest, resp *TTSResponse, err error)

	// WrapStream wraps a TTS audio stream for observability.
	// The wrapper should track bytes read and call AfterSynthesize on EOF/error.
	WrapStream(ctx context.Context, info TTSCallInfo, req TTSRequest, stream <-chan []byte) <-chan []byte
}

// STTCallInfo contains information about an STT operation.
type STTCallInfo struct {
	// CallID is a unique identifier for correlating BeforeRequest/AfterResponse.
	CallID string

	// Provider is the STT provider name (e.g., "deepgram", "openai").
	Provider string

	// StartTime is when the operation started.
	StartTime time.Time

	// Model is the STT model being used.
	Model string

	// Language is the expected language.
	Language string
}

// STTRequest contains the STT request details.
type STTRequest struct {
	// AudioSize is the size of the audio in bytes (for non-streaming).
	AudioSize int64

	// Encoding is the audio encoding (e.g., "pcm", "mulaw", "mp3").
	Encoding string

	// SampleRate is the audio sample rate.
	SampleRate int

	// Channels is the number of audio channels.
	Channels int

	// IsStreaming indicates if this is a streaming transcription.
	IsStreaming bool
}

// STTResponse contains the STT response details.
type STTResponse struct {
	// Transcript is the transcribed text.
	Transcript string

	// TranscriptLength is the character count.
	TranscriptLength int

	// Confidence is the transcription confidence (0-1).
	Confidence float64

	// AudioDuration is the duration of audio processed.
	AudioDuration time.Duration

	// Latency is the processing latency.
	Latency time.Duration

	// IsFinal indicates if this is a final transcription result.
	IsFinal bool
}

// STTHook provides observability for STT operations.
type STTHook interface {
	// BeforeTranscribe is called before STT transcription.
	// Returns a potentially modified context for propagation.
	BeforeTranscribe(ctx context.Context, info STTCallInfo, req STTRequest) context.Context

	// AfterTranscribe is called after STT transcription completes.
	AfterTranscribe(ctx context.Context, info STTCallInfo, req STTRequest, resp *STTResponse, err error)

	// WrapStreamWriter wraps an STT audio writer for observability.
	// The wrapper should track bytes written.
	WrapStreamWriter(ctx context.Context, info STTCallInfo, req STTRequest, writer io.WriteCloser) io.WriteCloser

	// OnStreamResult is called for each streaming transcription result.
	OnStreamResult(ctx context.Context, info STTCallInfo, resp STTResponse)
}

// NoOpTTSHook is a TTS hook that does nothing.
type NoOpTTSHook struct{}

func (NoOpTTSHook) BeforeSynthesize(ctx context.Context, _ TTSCallInfo, _ TTSRequest) context.Context {
	return ctx
}

func (NoOpTTSHook) AfterSynthesize(_ context.Context, _ TTSCallInfo, _ TTSRequest, _ *TTSResponse, _ error) {
}

func (NoOpTTSHook) WrapStream(_ context.Context, _ TTSCallInfo, _ TTSRequest, stream <-chan []byte) <-chan []byte {
	return stream
}

// NoOpSTTHook is an STT hook that does nothing.
type NoOpSTTHook struct{}

func (NoOpSTTHook) BeforeTranscribe(ctx context.Context, _ STTCallInfo, _ STTRequest) context.Context {
	return ctx
}

func (NoOpSTTHook) AfterTranscribe(_ context.Context, _ STTCallInfo, _ STTRequest, _ *STTResponse, _ error) {
}

func (NoOpSTTHook) WrapStreamWriter(_ context.Context, _ STTCallInfo, _ STTRequest, writer io.WriteCloser) io.WriteCloser {
	return writer
}

func (NoOpSTTHook) OnStreamResult(_ context.Context, _ STTCallInfo, _ STTResponse) {
}

// Verify interface compliance.
var (
	_ TTSHook = NoOpTTSHook{}
	_ STTHook = NoOpSTTHook{}
)
