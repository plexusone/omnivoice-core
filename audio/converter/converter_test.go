package converter

import (
	"testing"

	"github.com/plexusone/omnivoice-core/audio/codec"
	"github.com/plexusone/omnivoice-core/audio/format"
)

func TestConverter_Convert_SameFormat(t *testing.T) {
	conv := New()

	// PCM16 24kHz → PCM16 24kHz (no conversion needed)
	input := codec.Int16ToBytes([]int16{100, 200, 300, 400}, false)
	output, err := conv.Convert(input, format.OpenAI, format.OpenAI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != len(input) {
		t.Errorf("expected length %d, got %d", len(input), len(output))
	}
}

func TestConverter_Convert_MulawToPCM(t *testing.T) {
	conv := New()

	// Create mulaw samples
	pcmSamples := []int16{1000, 2000, 3000, 4000}
	mulaw := codec.MulawEncode(pcmSamples)

	// Convert mulaw 8kHz → PCM16 8kHz (same sample rate, just codec change)
	format8kMulaw := format.AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}
	format8kPCM := format.AudioFormat{Encoding: "pcm16", SampleRate: 8000, Channels: 1}

	output, err := conv.Convert(mulaw, format8kMulaw, format8kPCM)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be PCM16 bytes (2 bytes per sample)
	if len(output) != len(mulaw)*2 {
		t.Errorf("expected length %d, got %d", len(mulaw)*2, len(output))
	}
}

func TestConverter_Convert_PCMToMulaw(t *testing.T) {
	conv := New()

	// Create PCM16 samples
	pcmSamples := []int16{1000, 2000, 3000, 4000}
	pcmBytes := codec.Int16ToBytes(pcmSamples, false)

	// Convert PCM16 8kHz → mulaw 8kHz
	format8kPCM := format.AudioFormat{Encoding: "pcm16", SampleRate: 8000, Channels: 1}
	format8kMulaw := format.AudioFormat{Encoding: "mulaw", SampleRate: 8000, Channels: 1}

	output, err := conv.Convert(pcmBytes, format8kPCM, format8kMulaw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be mulaw bytes (1 byte per sample)
	if len(output) != len(pcmSamples) {
		t.Errorf("expected length %d, got %d", len(pcmSamples), len(output))
	}
}

func TestConverter_Convert_Resample(t *testing.T) {
	conv := New()

	// Create 80 samples at 8kHz (10ms of audio)
	samples8k := make([]int16, 80)
	for i := range samples8k {
		samples8k[i] = int16(i * 100)
	}
	pcm8k := codec.Int16ToBytes(samples8k, false)

	// Convert PCM16 8kHz → PCM16 24kHz (3x upsample)
	format8k := format.AudioFormat{Encoding: "pcm16", SampleRate: 8000, Channels: 1}
	format24k := format.AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}

	output, err := conv.Convert(pcm8k, format8k, format24k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be ~3x the input length (240 samples = 480 bytes)
	expectedLen := 240 * 2 // 240 samples * 2 bytes/sample
	if len(output) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(output))
	}
}

func TestConverter_Convert_Downsample(t *testing.T) {
	conv := New()

	// Create 240 samples at 24kHz (10ms of audio)
	samples24k := make([]int16, 240)
	for i := range samples24k {
		samples24k[i] = int16(i * 100)
	}
	pcm24k := codec.Int16ToBytes(samples24k, false)

	// Convert PCM16 24kHz → PCM16 8kHz (3x downsample)
	format24k := format.AudioFormat{Encoding: "pcm16", SampleRate: 24000, Channels: 1}
	format8k := format.AudioFormat{Encoding: "pcm16", SampleRate: 8000, Channels: 1}

	output, err := conv.Convert(pcm24k, format24k, format8k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be ~1/3 the input length (80 samples = 160 bytes)
	expectedLen := 80 * 2 // 80 samples * 2 bytes/sample
	if len(output) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(output))
	}
}

func TestConverter_TwilioToOpenAI(t *testing.T) {
	// Create mulaw samples (Twilio format)
	pcmSamples := make([]int16, 80) // 10ms at 8kHz
	for i := range pcmSamples {
		pcmSamples[i] = int16(i * 100)
	}
	mulaw := codec.MulawEncode(pcmSamples)

	// Convert Twilio → OpenAI
	output, err := TwilioToOpenAI(mulaw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be PCM16 24kHz (240 samples = 480 bytes)
	expectedLen := 240 * 2
	if len(output) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(output))
	}
}

func TestConverter_OpenAIToTwilio(t *testing.T) {
	// Create PCM16 24kHz samples (OpenAI format)
	samples := make([]int16, 240) // 10ms at 24kHz
	for i := range samples {
		samples[i] = int16(i * 100)
	}
	pcm24k := codec.Int16ToBytes(samples, false)

	// Convert OpenAI → Twilio
	output, err := OpenAIToTwilio(pcm24k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be mulaw 8kHz (80 samples = 80 bytes)
	expectedLen := 80
	if len(output) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(output))
	}
}

func TestConverter_TwilioToGemini(t *testing.T) {
	// Create mulaw samples (Twilio format)
	pcmSamples := make([]int16, 80) // 10ms at 8kHz
	for i := range pcmSamples {
		pcmSamples[i] = int16(i * 100)
	}
	mulaw := codec.MulawEncode(pcmSamples)

	// Convert Twilio → Gemini
	output, err := TwilioToGemini(mulaw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be PCM16 16kHz (160 samples = 320 bytes)
	expectedLen := 160 * 2
	if len(output) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(output))
	}
}

func TestConverter_GeminiToTwilio(t *testing.T) {
	// Create PCM16 24kHz samples (Gemini output format)
	samples := make([]int16, 240) // 10ms at 24kHz
	for i := range samples {
		samples[i] = int16(i * 100)
	}
	pcm24k := codec.Int16ToBytes(samples, false)

	// Convert Gemini → Twilio
	output, err := GeminiToTwilio(pcm24k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be mulaw 8kHz (80 samples = 80 bytes)
	expectedLen := 80
	if len(output) != expectedLen {
		t.Errorf("expected length %d, got %d", expectedLen, len(output))
	}
}

func TestConverter_EmptyInput(t *testing.T) {
	conv := New()

	output, err := conv.Convert(nil, format.Twilio, format.OpenAI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("expected empty output, got %d bytes", len(output))
	}

	output, err = conv.Convert([]byte{}, format.Twilio, format.OpenAI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("expected empty output, got %d bytes", len(output))
	}
}

func TestConverter_UnsupportedEncoding(t *testing.T) {
	conv := New()

	unsupported := format.AudioFormat{Encoding: "opus", SampleRate: 48000, Channels: 1}

	_, err := conv.Convert([]byte{1, 2, 3, 4}, unsupported, format.OpenAI)
	if err != ErrUnsupportedEncoding {
		t.Errorf("expected ErrUnsupportedEncoding, got %v", err)
	}

	_, err = conv.Convert([]byte{1, 2, 3, 4}, format.OpenAI, unsupported)
	if err != ErrUnsupportedEncoding {
		t.Errorf("expected ErrUnsupportedEncoding, got %v", err)
	}
}

func TestStreamConverter(t *testing.T) {
	sc := NewStreamConverter(format.Twilio, format.OpenAI)

	// Convert multiple chunks
	for i := 0; i < 10; i++ {
		pcmSamples := make([]int16, 80) // 10ms at 8kHz
		for j := range pcmSamples {
			pcmSamples[j] = int16(j * 100)
		}
		mulaw := codec.MulawEncode(pcmSamples)

		output, err := sc.Convert(mulaw)
		if err != nil {
			t.Fatalf("chunk %d: unexpected error: %v", i, err)
		}

		expectedLen := 240 * 2 // 10ms at 24kHz
		if len(output) != expectedLen {
			t.Errorf("chunk %d: expected length %d, got %d", i, expectedLen, len(output))
		}
	}
}
