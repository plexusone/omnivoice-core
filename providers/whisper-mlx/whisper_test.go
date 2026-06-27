package whispermlx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/plexusone/omnivoice-core/stt"
)

// getTestEndpoint returns the endpoint for testing.
// Uses WHISPER_ENDPOINT env var if set, otherwise the default endpoint.
func getTestEndpoint() string {
	if endpoint := os.Getenv("WHISPER_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return DefaultEndpoint
}

// skipIfNoServer skips the test if the Whisper server is not available.
func skipIfNoServer(t *testing.T, p *Provider) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := p.Health(ctx)
	if err != nil {
		t.Skipf("Skipping integration test: Whisper server not available at %s: %v", p.endpoint, err)
	}
}

// TestWhisper_New tests provider creation.
func TestWhisper_New(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()

	if p.Name() != ProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), ProviderName)
	}
}

// TestWhisperIntegration_Health tests the health check endpoint.
func TestWhisperIntegration_Health(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := p.Health(ctx)
	if err != nil {
		t.Fatalf("Health() failed: %v", err)
	}

	t.Logf("Health status:")
	t.Logf("  Healthy: %v", health.Healthy)
	t.Logf("  Model loaded: %v", health.ModelLoaded)
	t.Logf("  Model name: %s", health.ModelName)
	t.Logf("  Model version: %s", health.ModelVersion)
	t.Logf("  Supported languages: %v", health.SupportedLanguages)

	if !health.Healthy {
		t.Error("Expected server to be healthy")
	}
}

// TestWhisperIntegration_RuntimeInfo tests the runtime info endpoint.
func TestWhisperIntegration_RuntimeInfo(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := p.RuntimeInfo(ctx)
	if err != nil {
		t.Fatalf("RuntimeInfo() failed: %v", err)
	}

	t.Logf("Runtime info:")
	t.Logf("  Device type: %s", info.DeviceType)
	t.Logf("  Memory used: %d MB", info.MemoryUsedMB)
	t.Logf("  Memory available: %d MB", info.MemoryAvailableMB)
	t.Logf("  Framework version: %s", info.FrameworkVersion)
	t.Logf("  Python version: %s", info.PythonVersion)

	if info.ModelInfo != nil {
		t.Logf("  Model info:")
		t.Logf("    Name: %s", info.ModelInfo.Name)
		t.Logf("    Variant: %s", info.ModelInfo.Variant)
		t.Logf("    Parameters: %d", info.ModelInfo.ParameterCount)
		t.Logf("    Languages: %v", info.ModelInfo.SupportedLanguages)
	}

	if info.DeviceType == "" {
		t.Error("Expected device type to be set")
	}
}

// TestWhisperIntegration_ListModels tests model listing.
func TestWhisperIntegration_ListModels(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := p.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() failed: %v", err)
	}

	t.Logf("Available models: %d", len(models))
	for _, m := range models {
		t.Logf("  - %s: %s (%d MB, downloaded: %v)", m.Name, m.Description, m.SizeMB, m.IsDownloaded)
	}
}

// TestWhisperIntegration_Transcribe tests basic transcription.
// This test requires an audio file.
func TestWhisperIntegration_Transcribe(t *testing.T) {
	audioPath := os.Getenv("WHISPER_TEST_AUDIO")
	if audioPath == "" {
		t.Skip("Skipping transcription test: WHISPER_TEST_AUDIO not set")
	}

	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	// Read audio file
	audio, err := os.ReadFile(audioPath) //nolint:gosec // Path from env var is intentional
	if err != nil {
		t.Fatalf("Failed to read audio file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Ensure model is loaded
	health, err := p.Health(ctx)
	if err != nil {
		t.Fatalf("Health() failed: %v", err)
	}
	if !health.ModelLoaded {
		loadResult, err := p.LoadModel(ctx, "")
		if err != nil {
			t.Fatalf("LoadModel() failed: %v", err)
		}
		if !loadResult.Success {
			t.Fatalf("LoadModel() not successful: %s", loadResult.Error)
		}
		t.Logf("Model loaded: %s in %d ms", loadResult.ModelName, loadResult.LoadTimeMs)
	}

	// Test transcription
	config := stt.TranscriptionConfig{
		EnableWordTimestamps: true,
		EnablePunctuation:    true,
	}

	result, err := p.Transcribe(ctx, audio, config)
	if err != nil {
		t.Fatalf("Transcribe() failed: %v", err)
	}

	t.Logf("Transcription result:")
	t.Logf("  Text: %s", result.Text)
	t.Logf("  Language: %s (confidence: %.2f)", result.Language, result.LanguageConfidence)
	t.Logf("  Duration: %v", result.Duration)
	t.Logf("  Segments: %d", len(result.Segments))

	for i, seg := range result.Segments {
		t.Logf("  Segment %d [%v - %v]: %s", i, seg.StartTime, seg.EndTime, seg.Text)
		if len(seg.Words) > 0 {
			t.Logf("    Words: %d", len(seg.Words))
		}
	}

	if result.Text == "" {
		t.Error("Expected non-empty transcription text")
	}
}

// TestWhisperIntegration_ModelManagement tests model load/unload.
func TestWhisperIntegration_ModelManagement(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Test load model
	loadResult, err := p.LoadModel(ctx, "")
	if err != nil {
		t.Fatalf("LoadModel() failed: %v", err)
	}

	t.Logf("Load result:")
	t.Logf("  Success: %v", loadResult.Success)
	t.Logf("  Model: %s", loadResult.ModelName)
	t.Logf("  Load time: %d ms", loadResult.LoadTimeMs)
	t.Logf("  Memory used: %d MB", loadResult.MemoryUsedMB)
	if loadResult.Error != "" {
		t.Logf("  Error: %s", loadResult.Error)
	}

	if !loadResult.Success {
		t.Errorf("LoadModel() not successful: %s", loadResult.Error)
	}

	// Test unload model (optional - may want to keep loaded for other tests)
	if os.Getenv("WHISPER_TEST_UNLOAD") != "" {
		unloadResult, err := p.UnloadModel(ctx)
		if err != nil {
			t.Fatalf("UnloadModel() failed: %v", err)
		}

		t.Logf("Unload result:")
		t.Logf("  Success: %v", unloadResult.Success)
		t.Logf("  Memory freed: %d MB", unloadResult.MemoryFreedMB)

		if !unloadResult.Success {
			t.Error("UnloadModel() not successful")
		}
	}
}
