// Package f5tts provides an F5-TTS MLX provider for OmniVoice.
//
// F5-TTS is a high-quality text-to-speech model that supports zero-shot
// voice cloning from reference audio. This provider communicates with a
// local Python/MLX server via gRPC over Unix Domain Socket.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnivoice"
//	    _ "github.com/plexusone/omnivoice-core/providers/f5tts"
//	)
//
//	provider, err := omnivoice.GetTTSProvider("f5tts")
//	result, err := provider.Synthesize(ctx, "Hello world", tts.SynthesisConfig{})
package f5tts

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	omnivoice "github.com/plexusone/omnivoice-core"
	pb "github.com/plexusone/omnivoice-core/proto/localvoice/v1"
	"github.com/plexusone/omnivoice-core/registry"
	"github.com/plexusone/omnivoice-core/tts"
)

const (
	// ProviderName is the registered name for this provider.
	ProviderName = "f5tts"

	// DefaultEndpoint is the default gRPC endpoint for the F5-TTS server.
	DefaultEndpoint = "unix:///tmp/omnivoice-f5tts.sock"

	// DefaultSampleRate is the default output sample rate.
	DefaultSampleRate = 24000
)

func init() {
	omnivoice.RegisterTTSProvider(ProviderName, NewProvider, omnivoice.PriorityThick)
}

// Provider implements the TTS provider interface for F5-TTS MLX.
// It also implements several capability interfaces for local providers.
type Provider struct {
	conn   *grpc.ClientConn
	client pb.LocalVoiceClient

	endpoint string
}

// Compile-time interface checks
var (
	_ tts.Provider             = (*Provider)(nil)
	_ tts.VoiceCloner          = (*Provider)(nil)
	_ tts.ReferenceSynthesizer = (*Provider)(nil)
	_ tts.ProfileCacher        = (*Provider)(nil)
	_ tts.ModelManager         = (*Provider)(nil)
	_ tts.RuntimeChecker       = (*Provider)(nil)
	_ tts.HealthChecker        = (*Provider)(nil)
	_ io.Closer                = (*Provider)(nil)
)

// NewProvider creates a new F5-TTS provider from the registry config.
func NewProvider(cfg registry.ProviderConfig) (tts.Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	return New(endpoint)
}

// New creates a new F5-TTS provider with the given endpoint.
func New(endpoint string) (*Provider, error) {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to connect to %s: %w", endpoint, err)
	}

	return &Provider{
		conn:     conn,
		client:   pb.NewLocalVoiceClient(conn),
		endpoint: endpoint,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Synthesize converts text to speech and returns audio data.
func (p *Provider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
	req := &pb.SynthesizeRequest{
		Text:    text,
		VoiceId: config.VoiceID,
		Format:  toProtoFormat(config.OutputFormat),
	}
	if config.Speed > 0 {
		speed := float32(config.Speed)
		req.Speed = &speed
	}

	stream, err := p.client.Synthesize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("f5tts: synthesis failed: %w", err)
	}

	// Collect all audio chunks
	var audio []byte
	var metadata *pb.AudioMetadata

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("f5tts: stream error: %w", err)
		}

		audio = append(audio, chunk.Data...)

		if chunk.Metadata != nil && metadata == nil {
			metadata = chunk.Metadata
		}

		if chunk.IsFinal {
			break
		}
	}

	sampleRate := DefaultSampleRate
	format := config.OutputFormat
	if metadata != nil {
		sampleRate = int(metadata.SampleRate)
		format = fromProtoFormat(metadata.Format)
	}

	return &tts.SynthesisResult{
		Audio:          audio,
		Format:         format,
		SampleRate:     sampleRate,
		CharacterCount: len(text),
	}, nil
}

// SynthesizeStream converts text to speech with streaming output.
func (p *Provider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
	req := &pb.SynthesizeRequest{
		Text:    text,
		VoiceId: config.VoiceID,
		Format:  toProtoFormat(config.OutputFormat),
	}
	if config.Speed > 0 {
		speed := float32(config.Speed)
		req.Speed = &speed
	}

	stream, err := p.client.Synthesize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("f5tts: synthesis failed: %w", err)
	}

	ch := make(chan tts.StreamChunk)
	go func() {
		defer close(ch)
		for {
			chunk, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				ch <- tts.StreamChunk{Error: err}
				return
			}

			ch <- tts.StreamChunk{
				Audio:   chunk.Data,
				IsFinal: chunk.IsFinal,
			}

			if chunk.IsFinal {
				return
			}
		}
	}()

	return ch, nil
}

// ListVoices returns available voices from this provider.
func (p *Provider) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	health, err := p.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to list voices: %w", err)
	}

	voices := make([]tts.Voice, 0, len(health.AvailableVoices)+1)

	// Add default voice
	voices = append(voices, tts.Voice{
		ID:       "default",
		Name:     "Default",
		Provider: ProviderName,
	})

	// Add prepared profiles
	for _, id := range health.AvailableVoices {
		voices = append(voices, tts.Voice{
			ID:       id,
			Name:     id,
			Provider: ProviderName,
		})
	}

	return voices, nil
}

// GetVoice returns a specific voice by ID.
func (p *Provider) GetVoice(ctx context.Context, voiceID string) (*tts.Voice, error) {
	voices, err := p.ListVoices(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range voices {
		if v.ID == voiceID {
			return &v, nil
		}
	}

	return nil, tts.ErrVoiceNotFound
}

// CloneVoice creates a voice profile from reference audio.
func (p *Provider) CloneVoice(ctx context.Context, req tts.CloneVoiceRequest) (*tts.VoiceProfile, error) {
	profile, err := p.PrepareVoiceProfile(ctx, tts.PrepareVoiceProfileRequest{
		ProfileID:      req.Name,
		ReferenceAudio: req.ReferenceAudio,
		ReferenceText:  req.ReferenceText,
		Language:       req.Language,
	})
	if err != nil {
		return nil, err
	}

	return &tts.VoiceProfile{
		ID:            profile.ProfileID,
		Name:          req.Name,
		Language:      req.Language,
		CreatedAt:     profile.CreatedAt,
		Provider:      ProviderName,
		Cached:        profile.Cached,
		EmbeddingSize: profile.EmbeddingSizeBytes,
	}, nil
}

// SynthesizeWithReference performs synthesis using reference audio inline.
func (p *Provider) SynthesizeWithReference(ctx context.Context, req tts.ReferenceSynthesizeRequest) (*tts.SynthesisResult, error) {
	pbReq := &pb.ReferenceSynthesizeRequest{
		Text:           req.Text,
		ReferenceAudio: req.ReferenceAudio,
		ReferenceText:  req.ReferenceText,
		Format:         toProtoFormat(req.Config.OutputFormat),
	}
	if req.Config.Speed > 0 {
		speed := float32(req.Config.Speed)
		pbReq.Speed = &speed
	}

	stream, err := p.client.SynthesizeWithReference(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("f5tts: reference synthesis failed: %w", err)
	}

	// Collect all audio chunks
	var audio []byte
	var metadata *pb.AudioMetadata

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("f5tts: stream error: %w", err)
		}

		audio = append(audio, chunk.Data...)

		if chunk.Metadata != nil && metadata == nil {
			metadata = chunk.Metadata
		}

		if chunk.IsFinal {
			break
		}
	}

	sampleRate := DefaultSampleRate
	format := req.Config.OutputFormat
	if metadata != nil {
		sampleRate = int(metadata.SampleRate)
		format = fromProtoFormat(metadata.Format)
	}

	return &tts.SynthesisResult{
		Audio:          audio,
		Format:         format,
		SampleRate:     sampleRate,
		CharacterCount: len(req.Text),
	}, nil
}

// PrepareVoiceProfile pre-computes and caches a voice embedding.
func (p *Provider) PrepareVoiceProfile(ctx context.Context, req tts.PrepareVoiceProfileRequest) (*tts.PreparedProfile, error) {
	pbReq := &pb.PrepareVoiceProfileRequest{
		ProfileId:      req.ProfileID,
		ReferenceAudio: req.ReferenceAudio,
		ReferenceText:  req.ReferenceText,
	}
	if req.Language != "" {
		pbReq.Language = &req.Language
	}

	resp, err := p.client.PrepareVoiceProfile(ctx, pbReq)
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to prepare profile: %w", err)
	}

	return &tts.PreparedProfile{
		ProfileID:          resp.ProfileId,
		Cached:             resp.Cached,
		EmbeddingSizeBytes: resp.EmbeddingSizeBytes,
	}, nil
}

// ListPreparedProfiles returns all cached voice profiles.
func (p *Provider) ListPreparedProfiles(ctx context.Context) ([]*tts.PreparedProfile, error) {
	health, err := p.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to list profiles: %w", err)
	}

	profiles := make([]*tts.PreparedProfile, 0, len(health.AvailableVoices))
	for _, id := range health.AvailableVoices {
		profiles = append(profiles, &tts.PreparedProfile{
			ProfileID: id,
			Cached:    true,
		})
	}

	return profiles, nil
}

// DeletePreparedProfile removes a cached profile.
// Note: This is not currently supported by the proto - profiles persist until server restart.
func (p *Provider) DeletePreparedProfile(_ context.Context, _ string) error {
	return errors.New("f5tts: profile deletion not supported")
}

// LoadModel loads the TTS model into memory.
func (p *Provider) LoadModel(ctx context.Context) (*tts.LoadModelResult, error) {
	resp, err := p.client.LoadModel(ctx, &pb.LoadModelRequest{})
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to load model: %w", err)
	}

	result := &tts.LoadModelResult{
		Success:      resp.Success,
		LoadTimeMs:   resp.LoadTimeMs,
		MemoryUsedMB: resp.MemoryUsedMb,
	}
	if resp.ErrorMessage != nil {
		result.Error = *resp.ErrorMessage
	}

	return result, nil
}

// UnloadModel unloads the model from memory.
func (p *Provider) UnloadModel(ctx context.Context) (*tts.UnloadModelResult, error) {
	resp, err := p.client.UnloadModel(ctx, &pb.UnloadModelRequest{})
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to unload model: %w", err)
	}

	return &tts.UnloadModelResult{
		Success:       resp.Success,
		MemoryFreedMB: resp.MemoryFreedMb,
	}, nil
}

// IsModelLoaded returns whether the model is currently loaded.
func (p *Provider) IsModelLoaded() bool {
	ctx := context.Background()
	health, err := p.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return false
	}
	return health.ModelLoaded
}

// RuntimeInfo returns information about the runtime environment.
func (p *Provider) RuntimeInfo(ctx context.Context) (*tts.RuntimeInfo, error) {
	resp, err := p.client.RuntimeInfo(ctx, &pb.RuntimeInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("f5tts: failed to get runtime info: %w", err)
	}

	info := &tts.RuntimeInfo{
		DeviceType:        resp.DeviceType,
		MemoryUsedMB:      resp.MemoryUsedMb,
		MemoryAvailableMB: resp.MemoryAvailableMb,
		FrameworkVersion:  resp.FrameworkVersion,
		PythonVersion:     resp.PythonVersion,
	}

	if resp.ModelInfo != nil {
		info.ModelInfo = &tts.ModelInfo{
			Name:               resp.ModelInfo.Name,
			Version:            resp.ModelInfo.Version,
			ParameterCount:     resp.ModelInfo.ParameterCount,
			SupportedLanguages: resp.ModelInfo.SupportedLanguages,
		}
	}

	return info, nil
}

// Health returns the health status of the provider.
func (p *Provider) Health(ctx context.Context) (*tts.HealthStatus, error) {
	resp, err := p.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return nil, fmt.Errorf("f5tts: health check failed: %w", err)
	}

	return &tts.HealthStatus{
		Healthy:         resp.Healthy,
		ModelLoaded:     resp.ModelLoaded,
		ModelName:       resp.ModelName,
		ModelVersion:    resp.ModelVersion,
		AvailableVoices: resp.AvailableVoices,
	}, nil
}

// Close closes the gRPC connection.
func (p *Provider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// toProtoFormat converts a string format to proto AudioFormat.
func toProtoFormat(format string) pb.AudioFormat {
	switch format {
	case "wav":
		return pb.AudioFormat_AUDIO_FORMAT_WAV
	case "pcm", "pcm_s16le":
		return pb.AudioFormat_AUDIO_FORMAT_PCM_S16LE
	case "pcm_f32le":
		return pb.AudioFormat_AUDIO_FORMAT_PCM_F32LE
	case "mp3":
		return pb.AudioFormat_AUDIO_FORMAT_MP3
	case "mulaw", "ulaw_8000":
		return pb.AudioFormat_AUDIO_FORMAT_MULAW_8K
	case "opus":
		return pb.AudioFormat_AUDIO_FORMAT_OPUS
	default:
		return pb.AudioFormat_AUDIO_FORMAT_WAV
	}
}

// fromProtoFormat converts proto AudioFormat to string.
func fromProtoFormat(format pb.AudioFormat) string {
	switch format {
	case pb.AudioFormat_AUDIO_FORMAT_WAV:
		return "wav"
	case pb.AudioFormat_AUDIO_FORMAT_PCM_S16LE:
		return "pcm_s16le"
	case pb.AudioFormat_AUDIO_FORMAT_PCM_F32LE:
		return "pcm_f32le"
	case pb.AudioFormat_AUDIO_FORMAT_MP3:
		return "mp3"
	case pb.AudioFormat_AUDIO_FORMAT_MULAW_8K:
		return "ulaw_8000"
	case pb.AudioFormat_AUDIO_FORMAT_OPUS:
		return "opus"
	default:
		return "wav"
	}
}
