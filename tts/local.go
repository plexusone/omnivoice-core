// Package tts provides a unified interface for Text-to-Speech providers.
//
// This file defines capability interfaces for local TTS providers.
// Local providers (F5-TTS, Qwen3-TTS, Piper, etc.) may implement these
// interfaces to expose additional functionality beyond the base Provider.
//
// Use type assertions to check for capability support:
//
//	if cloner, ok := provider.(tts.VoiceCloner); ok {
//	    profile, err := cloner.CloneVoice(ctx, req)
//	}
package tts

import (
	"context"
	"io"
	"time"
)

// VoiceCloner is implemented by providers that support voice cloning.
// Voice cloning creates a voice profile from reference audio that can
// be used for subsequent synthesis.
type VoiceCloner interface {
	// CloneVoice creates a voice profile from reference audio.
	// The reference audio should be 5-15 seconds of clear speech.
	// The reference text must accurately transcribe the audio.
	CloneVoice(ctx context.Context, req CloneVoiceRequest) (*VoiceProfile, error)
}

// CloneVoiceRequest contains parameters for voice cloning.
type CloneVoiceRequest struct {
	// Name is a human-readable name for the voice profile.
	Name string

	// ReferenceAudio is the audio data (WAV or PCM format).
	ReferenceAudio []byte

	// ReferenceText is the transcript of the reference audio.
	// Must accurately match what is spoken.
	ReferenceText string

	// Language is the BCP-47 language code (e.g., "en-US").
	Language string
}

// VoiceProfile represents a cloned voice that can be used for synthesis.
type VoiceProfile struct {
	// ID is the unique identifier for this profile.
	ID string

	// Name is the human-readable name.
	Name string

	// Language is the BCP-47 language code.
	Language string

	// CreatedAt is when the profile was created.
	CreatedAt time.Time

	// Provider is the provider that created this profile.
	Provider string

	// Cached indicates whether the embedding is cached.
	Cached bool

	// EmbeddingSize is the size of the cached embedding in bytes.
	EmbeddingSize int64
}

// ReferenceSynthesizer is implemented by providers that support
// zero-shot synthesis with reference audio (without pre-creating a profile).
type ReferenceSynthesizer interface {
	// SynthesizeWithReference performs synthesis using reference audio inline.
	// This is useful for one-off synthesis without creating a persistent profile.
	SynthesizeWithReference(ctx context.Context, req ReferenceSynthesizeRequest) (*SynthesisResult, error)
}

// ReferenceSynthesizeRequest contains parameters for reference-based synthesis.
type ReferenceSynthesizeRequest struct {
	// Text is the text to synthesize.
	Text string

	// ReferenceAudio is the reference speaker's audio (WAV or PCM).
	ReferenceAudio []byte

	// ReferenceText is the transcript of the reference audio.
	ReferenceText string

	// Config contains synthesis configuration (format, speed, etc.).
	Config SynthesisConfig
}

// StreamingReferenceSynthesizer extends ReferenceSynthesizer with streaming output.
type StreamingReferenceSynthesizer interface {
	ReferenceSynthesizer

	// SynthesizeWithReferenceStream performs streaming synthesis with reference audio.
	SynthesizeWithReferenceStream(ctx context.Context, req ReferenceSynthesizeRequest) (<-chan StreamChunk, error)
}

// ProfileCacher is implemented by providers that support pre-computing
// voice embeddings for faster subsequent synthesis.
type ProfileCacher interface {
	// PrepareVoiceProfile pre-computes and caches a voice embedding.
	// Subsequent synthesis using this profile ID will be faster.
	PrepareVoiceProfile(ctx context.Context, req PrepareVoiceProfileRequest) (*PreparedProfile, error)

	// ListPreparedProfiles returns all cached voice profiles.
	ListPreparedProfiles(ctx context.Context) ([]*PreparedProfile, error)

	// DeletePreparedProfile removes a cached profile.
	DeletePreparedProfile(ctx context.Context, profileID string) error
}

// PrepareVoiceProfileRequest contains parameters for profile preparation.
type PrepareVoiceProfileRequest struct {
	// ProfileID is the unique identifier for this profile.
	ProfileID string

	// ReferenceAudio is the reference speaker's audio.
	ReferenceAudio []byte

	// ReferenceText is the transcript of the reference audio.
	ReferenceText string

	// Language is the BCP-47 language code.
	Language string
}

// PreparedProfile represents a cached voice embedding.
type PreparedProfile struct {
	// ProfileID is the unique identifier.
	ProfileID string

	// Cached indicates whether caching succeeded.
	Cached bool

	// EmbeddingSizeBytes is the size of the cached embedding.
	EmbeddingSizeBytes int64

	// CreatedAt is when the profile was prepared.
	CreatedAt time.Time
}

// ModelManager is implemented by providers that support explicit
// model loading and unloading.
type ModelManager interface {
	// LoadModel loads the TTS model into memory.
	LoadModel(ctx context.Context) (*LoadModelResult, error)

	// UnloadModel unloads the model from memory.
	UnloadModel(ctx context.Context) (*UnloadModelResult, error)

	// IsModelLoaded returns whether the model is currently loaded.
	IsModelLoaded() bool
}

// LoadModelResult contains the result of loading a model.
type LoadModelResult struct {
	// Success indicates whether loading succeeded.
	Success bool

	// LoadTimeMs is the time taken to load in milliseconds.
	LoadTimeMs int64

	// MemoryUsedMB is the memory consumed by the model.
	MemoryUsedMB int64

	// Error contains error details if Success is false.
	Error string
}

// UnloadModelResult contains the result of unloading a model.
type UnloadModelResult struct {
	// Success indicates whether unloading succeeded.
	Success bool

	// MemoryFreedMB is the memory freed by unloading.
	MemoryFreedMB int64
}

// RuntimeChecker is implemented by providers that can report
// runtime environment information.
type RuntimeChecker interface {
	// RuntimeInfo returns information about the runtime environment.
	RuntimeInfo(ctx context.Context) (*RuntimeInfo, error)
}

// RuntimeInfo contains details about the runtime environment.
type RuntimeInfo struct {
	// DeviceType is the compute device ("mlx", "mps", "cpu", "cuda").
	DeviceType string

	// MemoryUsedMB is the current memory usage.
	MemoryUsedMB int64

	// MemoryAvailableMB is the available memory.
	MemoryAvailableMB int64

	// FrameworkVersion is the ML framework version (e.g., MLX version).
	FrameworkVersion string

	// PythonVersion is the Python version if applicable.
	PythonVersion string

	// ModelInfo contains loaded model information.
	ModelInfo *ModelInfo
}

// ModelInfo contains details about a loaded model.
type ModelInfo struct {
	// Name is the model name.
	Name string

	// Version is the model version.
	Version string

	// ParameterCount is the number of model parameters.
	ParameterCount int64

	// SupportedLanguages lists supported language codes.
	SupportedLanguages []string
}

// HealthChecker is implemented by providers that support health checks.
type HealthChecker interface {
	// Health returns the health status of the provider.
	Health(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus contains health information for a provider.
type HealthStatus struct {
	// Healthy indicates whether the provider is operational.
	Healthy bool

	// ModelLoaded indicates whether the model is loaded.
	ModelLoaded bool

	// ModelName is the name of the loaded model.
	ModelName string

	// ModelVersion is the version of the loaded model.
	ModelVersion string

	// AvailableVoices lists available voice profile IDs.
	AvailableVoices []string
}

// Closer is implemented by providers that hold resources requiring cleanup.
type Closer interface {
	io.Closer
}
