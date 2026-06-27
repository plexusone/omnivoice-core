// Package whispermlx provides a Whisper MLX provider for OmniVoice STT.
//
// Whisper MLX is OpenAI's Whisper model optimized for Apple Silicon using MLX.
// This provider communicates with a local Python/MLX server via gRPC over
// Unix Domain Socket.
//
// This package requires Apple Silicon (M1/M2/M3/M4) for MLX acceleration.
//
// Usage:
//
//	import (
//	    "github.com/plexusone/omnivoice"
//	    _ "github.com/plexusone/omnivoice-core/providers/whisper-mlx"
//	)
//
//	provider, err := omnivoice.GetSTTProvider("whisper-mlx")
//	result, err := provider.Transcribe(ctx, audioBytes, stt.TranscriptionConfig{})
package whispermlx

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	omnivoice "github.com/plexusone/omnivoice-core"
	pb "github.com/plexusone/omnivoice-core/proto/localstt/v1"
	"github.com/plexusone/omnivoice-core/registry"
	"github.com/plexusone/omnivoice-core/stt"
)

const (
	// ProviderName is the registered name for this provider.
	ProviderName = "whisper-mlx"

	// DefaultEndpoint is the default gRPC endpoint for the Whisper server.
	DefaultEndpoint = "unix:///tmp/omnivoice-whisper.sock"

	// DefaultModel is the default Whisper model variant.
	DefaultModel = "large-v3-turbo"
)

func init() {
	omnivoice.RegisterSTTProvider(ProviderName, NewProvider, omnivoice.PriorityThick)
}

// Provider implements the STT provider interface for Whisper MLX.
type Provider struct {
	conn   *grpc.ClientConn
	client pb.LocalSTTClient

	endpoint string
}

// Compile-time interface checks
var (
	_ stt.Provider = (*Provider)(nil)
	_ io.Closer    = (*Provider)(nil)
)

// NewProvider creates a new Whisper MLX provider from the registry config.
func NewProvider(cfg registry.ProviderConfig) (stt.Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	return New(endpoint)
}

// New creates a new Whisper MLX provider with the given endpoint.
func New(endpoint string) (*Provider, error) {
	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: failed to connect to %s: %w", endpoint, err)
	}

	return &Provider{
		conn:     conn,
		client:   pb.NewLocalSTTClient(conn),
		endpoint: endpoint,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Transcribe converts audio to text.
func (p *Provider) Transcribe(ctx context.Context, audio []byte, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	pbConfig := &pb.TranscriptionConfig{
		EnableWordTimestamps: config.EnableWordTimestamps,
		EnablePunctuation:    config.EnablePunctuation,
	}

	if config.Language != "" {
		pbConfig.Language = &config.Language
	}
	if config.Model != "" {
		pbConfig.Model = &config.Model
	}

	// Set input format if provided
	if config.Encoding != "" || config.SampleRate > 0 {
		pbConfig.InputFormat = &pb.AudioFormat{
			Encoding:   config.Encoding,
			SampleRate: int32(config.SampleRate), //nolint:gosec // G115: sample rate is always within int32 range
			Channels:   int32(config.Channels),   //nolint:gosec // G115: channels is always within int32 range
		}
	}

	req := &pb.TranscribeRequest{
		Audio:  audio,
		Config: pbConfig,
	}

	resp, err := p.client.Transcribe(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: transcription failed: %w", err)
	}

	// Convert protobuf response to STT types
	result := &stt.TranscriptionResult{
		Text:               resp.Text,
		Language:           resp.Language,
		LanguageConfidence: float64(resp.LanguageConfidence),
		Duration:           time.Duration(resp.DurationMs) * time.Millisecond,
	}

	// Convert segments
	result.Segments = make([]stt.Segment, len(resp.Segments))
	for i, seg := range resp.Segments {
		result.Segments[i] = stt.Segment{
			Text:       seg.Text,
			StartTime:  time.Duration(seg.StartMs) * time.Millisecond,
			EndTime:    time.Duration(seg.EndMs) * time.Millisecond,
			Confidence: float64(seg.Confidence),
		}

		// Convert words if present
		if len(seg.Words) > 0 {
			result.Segments[i].Words = make([]stt.Word, len(seg.Words))
			for j, word := range seg.Words {
				result.Segments[i].Words[j] = stt.Word{
					Text:       word.Text,
					StartTime:  time.Duration(word.StartMs) * time.Millisecond,
					EndTime:    time.Duration(word.EndMs) * time.Millisecond,
					Confidence: float64(word.Confidence),
				}
			}
		}
	}

	return result, nil
}

// TranscribeFile transcribes audio from a file path.
// For local providers, this reads the file and calls Transcribe.
func (p *Provider) TranscribeFile(ctx context.Context, filePath string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	// Read the file and delegate to Transcribe
	// Note: For a production implementation, you might want to stream the file
	// or have the server read it directly.
	return nil, fmt.Errorf("whisper-mlx: TranscribeFile not implemented - use Transcribe with audio bytes")
}

// TranscribeURL transcribes audio from a URL.
// For local providers, this is typically not supported.
func (p *Provider) TranscribeURL(_ context.Context, _ string, _ stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return nil, fmt.Errorf("whisper-mlx: TranscribeURL not supported for local providers")
}

// Health returns the health status of the provider.
func (p *Provider) Health(ctx context.Context) (*HealthStatus, error) {
	resp, err := p.client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: health check failed: %w", err)
	}

	return &HealthStatus{
		Healthy:            resp.Healthy,
		ModelLoaded:        resp.ModelLoaded,
		ModelName:          resp.ModelName,
		ModelVersion:       resp.ModelVersion,
		SupportedLanguages: resp.SupportedLanguages,
	}, nil
}

// HealthStatus contains health information for the Whisper provider.
type HealthStatus struct {
	Healthy            bool
	ModelLoaded        bool
	ModelName          string
	ModelVersion       string
	SupportedLanguages []string
}

// LoadModel loads the STT model into memory.
func (p *Provider) LoadModel(ctx context.Context, model string) (*LoadModelResult, error) {
	req := &pb.LoadModelRequest{}
	if model != "" {
		req.Model = &model
	}

	resp, err := p.client.LoadModel(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: failed to load model: %w", err)
	}

	result := &LoadModelResult{
		Success:      resp.Success,
		LoadTimeMs:   resp.LoadTimeMs,
		MemoryUsedMB: resp.MemoryUsedMb,
		ModelName:    resp.ModelName,
	}
	if resp.ErrorMessage != nil {
		result.Error = *resp.ErrorMessage
	}

	return result, nil
}

// LoadModelResult contains the result of loading a model.
type LoadModelResult struct {
	Success      bool
	LoadTimeMs   int64
	MemoryUsedMB int64
	ModelName    string
	Error        string
}

// UnloadModel unloads the model from memory.
func (p *Provider) UnloadModel(ctx context.Context) (*UnloadModelResult, error) {
	resp, err := p.client.UnloadModel(ctx, &pb.UnloadModelRequest{})
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: failed to unload model: %w", err)
	}

	return &UnloadModelResult{
		Success:       resp.Success,
		MemoryFreedMB: resp.MemoryFreedMb,
	}, nil
}

// UnloadModelResult contains the result of unloading a model.
type UnloadModelResult struct {
	Success       bool
	MemoryFreedMB int64
}

// RuntimeInfo returns information about the runtime environment.
func (p *Provider) RuntimeInfo(ctx context.Context) (*RuntimeInfo, error) {
	resp, err := p.client.RuntimeInfo(ctx, &pb.RuntimeInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: failed to get runtime info: %w", err)
	}

	info := &RuntimeInfo{
		DeviceType:        resp.DeviceType,
		MemoryUsedMB:      resp.MemoryUsedMb,
		MemoryAvailableMB: resp.MemoryAvailableMb,
		FrameworkVersion:  resp.FrameworkVersion,
		PythonVersion:     resp.PythonVersion,
	}

	if resp.ModelInfo != nil {
		info.ModelInfo = &ModelInfo{
			Name:               resp.ModelInfo.Name,
			Variant:            resp.ModelInfo.Variant,
			ParameterCount:     resp.ModelInfo.ParameterCount,
			SupportedLanguages: resp.ModelInfo.SupportedLanguages,
		}
	}

	return info, nil
}

// RuntimeInfo contains runtime environment details.
type RuntimeInfo struct {
	DeviceType        string
	MemoryUsedMB      int64
	MemoryAvailableMB int64
	FrameworkVersion  string
	PythonVersion     string
	ModelInfo         *ModelInfo
}

// ModelInfo contains details about the loaded model.
type ModelInfo struct {
	Name               string
	Variant            string
	ParameterCount     int64
	SupportedLanguages []string
}

// ListModels returns available Whisper models.
func (p *Provider) ListModels(ctx context.Context) ([]AvailableModel, error) {
	resp, err := p.client.ListModels(ctx, &pb.ListModelsRequest{})
	if err != nil {
		return nil, fmt.Errorf("whisper-mlx: failed to list models: %w", err)
	}

	models := make([]AvailableModel, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = AvailableModel{
			Name:         m.Name,
			Description:  m.Description,
			SizeMB:       m.SizeMb,
			IsDownloaded: m.IsDownloaded,
		}
	}

	return models, nil
}

// AvailableModel describes an available Whisper model.
type AvailableModel struct {
	Name         string
	Description  string
	SizeMB       int64
	IsDownloaded bool
}

// Close closes the gRPC connection.
func (p *Provider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
