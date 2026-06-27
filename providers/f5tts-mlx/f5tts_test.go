package f5ttsmlx

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/plexusone/omnivoice-core/tts"
)

// getTestEndpoint returns the endpoint for testing.
// Uses F5TTS_ENDPOINT env var if set, otherwise the default endpoint.
func getTestEndpoint() string {
	if endpoint := os.Getenv("F5TTS_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return DefaultEndpoint
}

// skipIfNoServer skips the test if the F5-TTS server is not available.
func skipIfNoServer(t *testing.T, p *Provider) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := p.Health(ctx)
	if err != nil {
		t.Skipf("Skipping integration test: F5-TTS server not available at %s: %v", p.endpoint, err)
	}
}

// TestF5TTS_New tests provider creation.
func TestF5TTS_New(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()

	if p.Name() != ProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), ProviderName)
	}
}

// TestF5TTSIntegration_Health tests the health check endpoint.
func TestF5TTSIntegration_Health(t *testing.T) {
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
	t.Logf("  Available voices: %v", health.AvailableVoices)

	if !health.Healthy {
		t.Error("Expected server to be healthy")
	}
}

// TestF5TTSIntegration_RuntimeInfo tests the runtime info endpoint.
func TestF5TTSIntegration_RuntimeInfo(t *testing.T) {
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
		t.Logf("    Version: %s", info.ModelInfo.Version)
		t.Logf("    Parameters: %d", info.ModelInfo.ParameterCount)
		t.Logf("    Languages: %v", info.ModelInfo.SupportedLanguages)
	}

	if info.DeviceType == "" {
		t.Error("Expected device type to be set")
	}
}

// TestF5TTSIntegration_Synthesize tests basic text-to-speech synthesis.
func TestF5TTSIntegration_Synthesize(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure model is loaded
	if !p.IsModelLoaded() {
		loadResult, err := p.LoadModel(ctx)
		if err != nil {
			t.Fatalf("LoadModel() failed: %v", err)
		}
		if !loadResult.Success {
			t.Fatalf("LoadModel() not successful: %s", loadResult.Error)
		}
		t.Logf("Model loaded in %d ms", loadResult.LoadTimeMs)
	}

	// Test synthesis
	config := tts.SynthesisConfig{
		OutputFormat: "wav",
	}

	result, err := p.Synthesize(ctx, "Hello world, this is a test.", config)
	if err != nil {
		t.Fatalf("Synthesize() failed: %v", err)
	}

	t.Logf("Synthesis result:")
	t.Logf("  Format: %s", result.Format)
	t.Logf("  Sample rate: %d", result.SampleRate)
	t.Logf("  Character count: %d", result.CharacterCount)
	t.Logf("  Audio size: %d bytes", len(result.Audio))

	if len(result.Audio) == 0 {
		t.Error("Expected non-empty audio data")
	}
	if result.SampleRate == 0 {
		t.Error("Expected non-zero sample rate")
	}
}

// TestF5TTSIntegration_SynthesizeStream tests streaming synthesis.
func TestF5TTSIntegration_SynthesizeStream(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure model is loaded
	if !p.IsModelLoaded() {
		if _, err := p.LoadModel(ctx); err != nil {
			t.Fatalf("LoadModel() failed: %v", err)
		}
	}

	config := tts.SynthesisConfig{
		OutputFormat: "wav",
	}

	ch, err := p.SynthesizeStream(ctx, "Testing streaming synthesis.", config)
	if err != nil {
		t.Fatalf("SynthesizeStream() failed: %v", err)
	}

	var totalSize int
	var chunkCount int

	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Stream chunk error: %v", chunk.Error)
		}

		chunkCount++
		totalSize += len(chunk.Audio)

		if chunk.IsFinal {
			t.Logf("Received final chunk")
		}
	}

	t.Logf("Received %d chunks, total %d bytes", chunkCount, totalSize)

	if chunkCount == 0 {
		t.Error("Expected at least one chunk")
	}
	if totalSize == 0 {
		t.Error("Expected non-empty audio data")
	}
}

// TestF5TTSIntegration_ListVoices tests voice listing.
func TestF5TTSIntegration_ListVoices(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	voices, err := p.ListVoices(ctx)
	if err != nil {
		t.Fatalf("ListVoices() failed: %v", err)
	}

	t.Logf("Found %d voices:", len(voices))
	for _, v := range voices {
		t.Logf("  - %s: %s (provider: %s)", v.ID, v.Name, v.Provider)
	}

	// Should at least have the default voice
	if len(voices) == 0 {
		t.Error("Expected at least one voice")
	}

	// Verify default voice exists
	hasDefault := false
	for _, v := range voices {
		if v.ID == "default" {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		t.Error("Expected default voice to be present")
	}
}

// TestF5TTSIntegration_GetVoice tests getting a specific voice.
func TestF5TTSIntegration_GetVoice(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test getting existing voice
	voice, err := p.GetVoice(ctx, "default")
	if err != nil {
		t.Fatalf("GetVoice(default) failed: %v", err)
	}
	if voice.ID != "default" {
		t.Errorf("GetVoice() ID = %q, want %q", voice.ID, "default")
	}

	// Test getting non-existent voice
	_, err = p.GetVoice(ctx, "nonexistent-voice-12345")
	if err == nil {
		t.Error("Expected error for non-existent voice")
	}
	if err != tts.ErrVoiceNotFound {
		t.Errorf("Expected ErrVoiceNotFound, got: %v", err)
	}
}

// TestF5TTSIntegration_ModelManagement tests model load/unload.
func TestF5TTSIntegration_ModelManagement(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Test load model
	loadResult, err := p.LoadModel(ctx)
	if err != nil {
		t.Fatalf("LoadModel() failed: %v", err)
	}

	t.Logf("Load result:")
	t.Logf("  Success: %v", loadResult.Success)
	t.Logf("  Load time: %d ms", loadResult.LoadTimeMs)
	t.Logf("  Memory used: %d MB", loadResult.MemoryUsedMB)
	if loadResult.Error != "" {
		t.Logf("  Error: %s", loadResult.Error)
	}

	if !loadResult.Success {
		t.Errorf("LoadModel() not successful: %s", loadResult.Error)
	}

	// Verify model is loaded
	if !p.IsModelLoaded() {
		t.Error("Expected model to be loaded after LoadModel()")
	}

	// Test unload model (optional - may want to keep loaded for other tests)
	if os.Getenv("F5TTS_TEST_UNLOAD") != "" {
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

// TestF5TTSIntegration_ListPreparedProfiles tests profile listing.
func TestF5TTSIntegration_ListPreparedProfiles(t *testing.T) {
	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	profiles, err := p.ListPreparedProfiles(ctx)
	if err != nil {
		t.Fatalf("ListPreparedProfiles() failed: %v", err)
	}

	t.Logf("Found %d prepared profiles:", len(profiles))
	for _, profile := range profiles {
		t.Logf("  - %s (cached: %v)", profile.ProfileID, profile.Cached)
	}
}

// TestF5TTSIntegration_PrepareVoiceProfile tests voice profile preparation.
// This test requires a reference audio file.
func TestF5TTSIntegration_PrepareVoiceProfile(t *testing.T) {
	refAudioPath := os.Getenv("F5TTS_TEST_REFERENCE_AUDIO")
	refText := os.Getenv("F5TTS_TEST_REFERENCE_TEXT")
	if refAudioPath == "" {
		t.Skip("Skipping profile test: F5TTS_TEST_REFERENCE_AUDIO not set")
	}

	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	// Read reference audio
	refAudio, err := os.ReadFile(refAudioPath) //nolint:gosec // Path from env var is intentional
	if err != nil {
		t.Fatalf("Failed to read reference audio: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure model is loaded
	if !p.IsModelLoaded() {
		if _, err := p.LoadModel(ctx); err != nil {
			t.Fatalf("LoadModel() failed: %v", err)
		}
	}

	// Prepare voice profile
	req := tts.PrepareVoiceProfileRequest{
		ProfileID:      "test-profile",
		ReferenceAudio: refAudio,
		ReferenceText:  refText,
		Language:       "en",
	}

	profile, err := p.PrepareVoiceProfile(ctx, req)
	if err != nil {
		t.Fatalf("PrepareVoiceProfile() failed: %v", err)
	}

	t.Logf("Prepared profile:")
	t.Logf("  ID: %s", profile.ProfileID)
	t.Logf("  Cached: %v", profile.Cached)
	t.Logf("  Embedding size: %d bytes", profile.EmbeddingSizeBytes)

	// Test synthesis with the prepared profile
	config := tts.SynthesisConfig{
		VoiceID:      profile.ProfileID,
		OutputFormat: "wav",
	}

	result, err := p.Synthesize(ctx, "This is a test using the cloned voice.", config)
	if err != nil {
		t.Fatalf("Synthesize with profile failed: %v", err)
	}

	t.Logf("Synthesis with profile: %d bytes", len(result.Audio))
}

// TestF5TTSIntegration_SynthesizeWithReference tests inline reference synthesis.
// This test requires a reference audio file.
func TestF5TTSIntegration_SynthesizeWithReference(t *testing.T) {
	refAudioPath := os.Getenv("F5TTS_TEST_REFERENCE_AUDIO")
	refText := os.Getenv("F5TTS_TEST_REFERENCE_TEXT")
	if refAudioPath == "" {
		t.Skip("Skipping reference synthesis test: F5TTS_TEST_REFERENCE_AUDIO not set")
	}

	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	// Read reference audio
	refAudio, err := os.ReadFile(refAudioPath) //nolint:gosec // Path from env var is intentional
	if err != nil {
		t.Fatalf("Failed to read reference audio: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Ensure model is loaded
	if !p.IsModelLoaded() {
		if _, err := p.LoadModel(ctx); err != nil {
			t.Fatalf("LoadModel() failed: %v", err)
		}
	}

	req := tts.ReferenceSynthesizeRequest{
		Text:           "This is synthesized with inline reference audio.",
		ReferenceAudio: refAudio,
		ReferenceText:  refText,
		Config: tts.SynthesisConfig{
			OutputFormat: "wav",
		},
	}

	result, err := p.SynthesizeWithReference(ctx, req)
	if err != nil {
		t.Fatalf("SynthesizeWithReference() failed: %v", err)
	}

	t.Logf("Reference synthesis result:")
	t.Logf("  Format: %s", result.Format)
	t.Logf("  Sample rate: %d", result.SampleRate)
	t.Logf("  Audio size: %d bytes", len(result.Audio))

	if len(result.Audio) == 0 {
		t.Error("Expected non-empty audio data")
	}
}

// TestF5TTSIntegration_CloneVoice tests the voice cloning interface.
// This test requires a reference audio file.
func TestF5TTSIntegration_CloneVoice(t *testing.T) {
	refAudioPath := os.Getenv("F5TTS_TEST_REFERENCE_AUDIO")
	refText := os.Getenv("F5TTS_TEST_REFERENCE_TEXT")
	if refAudioPath == "" {
		t.Skip("Skipping clone test: F5TTS_TEST_REFERENCE_AUDIO not set")
	}

	p, err := New(getTestEndpoint())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()
	skipIfNoServer(t, p)

	// Read reference audio
	refAudio, err := os.ReadFile(refAudioPath) //nolint:gosec // Path from env var is intentional
	if err != nil {
		t.Fatalf("Failed to read reference audio: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure model is loaded
	if !p.IsModelLoaded() {
		if _, err := p.LoadModel(ctx); err != nil {
			t.Fatalf("LoadModel() failed: %v", err)
		}
	}

	req := tts.CloneVoiceRequest{
		Name:           "test-clone",
		ReferenceAudio: refAudio,
		ReferenceText:  refText,
		Language:       "en",
	}

	profile, err := p.CloneVoice(ctx, req)
	if err != nil {
		t.Fatalf("CloneVoice() failed: %v", err)
	}

	t.Logf("Cloned voice profile:")
	t.Logf("  ID: %s", profile.ID)
	t.Logf("  Name: %s", profile.Name)
	t.Logf("  Provider: %s", profile.Provider)
	t.Logf("  Cached: %v", profile.Cached)
	t.Logf("  Embedding size: %d bytes", profile.EmbeddingSize)

	if profile.ID == "" {
		t.Error("Expected non-empty profile ID")
	}
	if profile.Provider != ProviderName {
		t.Errorf("Provider = %q, want %q", profile.Provider, ProviderName)
	}
}
