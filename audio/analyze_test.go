package audio

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeRMS(t *testing.T) {
	tests := []struct {
		name     string
		samples  []float64
		expected float64
	}{
		{
			name:     "silence",
			samples:  []float64{0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "constant",
			samples:  []float64{0.5, 0.5, 0.5, 0.5},
			expected: 0.5,
		},
		{
			name:     "sine wave peak",
			samples:  []float64{1, 0, -1, 0},
			expected: math.Sqrt(0.5), // sqrt(2/4)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeRMS(tt.samples)
			if math.Abs(got-tt.expected) > 0.0001 {
				t.Errorf("computeRMS() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScoreMetrics(t *testing.T) {
	// Good segment: high speech ratio, moderate energy, consistent
	good := SegmentMetrics{
		SpeechRatio:    0.75,
		MeanEnergy:     0.1,
		EnergyVariance: 0.01,
		SilenceGaps:    0,
		MaxSilenceGap:  0,
		PeakClipping:   0,
	}

	// Bad segment: low speech, silence gaps
	bad := SegmentMetrics{
		SpeechRatio:    0.3,
		MeanEnergy:     0.05,
		EnergyVariance: 0.1,
		SilenceGaps:    3,
		MaxSilenceGap:  2.0,
		PeakClipping:   0,
	}

	goodScore := scoreMetrics(good)
	badScore := scoreMetrics(bad)

	if goodScore <= badScore {
		t.Errorf("good segment score (%v) should be higher than bad segment score (%v)", goodScore, badScore)
	}
}

func TestBytesToSamples(t *testing.T) {
	// 16-bit stereo: two samples of (0.5, -0.5) mixed to mono = 0
	// 0.5 * 32768 = 16384 = 0x4000
	// -0.5 * 32768 = -16384 = 0xC000 (signed)
	data := []byte{
		0x00, 0x40, // Left: +16384
		0x00, 0xC0, // Right: -16384
		0x00, 0x40, // Left: +16384
		0x00, 0xC0, // Right: -16384
	}

	samples, err := bytesToSamples(data, 16, 2)
	if err != nil {
		t.Fatalf("bytesToSamples failed: %v", err)
	}

	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}

	// Mixed to mono: (0.5 + -0.5) / 2 = 0
	for i, s := range samples {
		if math.Abs(s) > 0.001 {
			t.Errorf("sample[%d] = %v, expected ~0", i, s)
		}
	}
}

func TestWriteAndReadWAV(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.wav")

	// Create test samples: 1 second of 440Hz sine wave at 24kHz
	sampleRate := uint32(24000)
	duration := 1.0
	numSamples := int(float64(sampleRate) * duration)
	samples := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*440*t)
	}

	// Write
	if err := WriteWAV(path, samples, sampleRate); err != nil {
		t.Fatalf("WriteWAV failed: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Expected size: 44 byte header + 2 bytes per sample
	expectedSize := int64(44 + numSamples*2)
	if info.Size() != expectedSize {
		t.Errorf("file size = %d, want %d", info.Size(), expectedSize)
	}

	// Read back
	header, readSamples, err := ReadWAV(path)
	if err != nil {
		t.Fatalf("ReadWAV failed: %v", err)
	}

	if header.SampleRate != sampleRate {
		t.Errorf("sample rate = %d, want %d", header.SampleRate, sampleRate)
	}
	if header.NumChannels != 1 {
		t.Errorf("num channels = %d, want 1", header.NumChannels)
	}
	if header.BitsPerSample != 16 {
		t.Errorf("bits per sample = %d, want 16", header.BitsPerSample)
	}
	if len(readSamples) != numSamples {
		t.Errorf("num samples = %d, want %d", len(readSamples), numSamples)
	}

	// Verify samples are close (some quantization error expected)
	maxError := 0.0
	for i := 0; i < numSamples; i++ {
		err := math.Abs(samples[i] - readSamples[i])
		if err > maxError {
			maxError = err
		}
	}

	// 16-bit quantization error should be < 1/32768 ≈ 0.00003
	if maxError > 0.0001 {
		t.Errorf("max sample error = %v, want < 0.0001", maxError)
	}
}

func TestFindBestSegments(t *testing.T) {
	sampleRate := uint32(24000)

	// Create 60 seconds of audio with varying characteristics
	totalDuration := 60.0
	numSamples := int(totalDuration * float64(sampleRate))
	samples := make([]float64, numSamples)

	// 0-20s: silence with occasional noise
	for i := 0; i < 20*int(sampleRate); i++ {
		if i%(int(sampleRate)/2) < 1000 {
			samples[i] = 0.3 * math.Sin(float64(i)*0.1)
		}
	}

	// 20-40s: consistent speech-like audio (should be selected)
	for i := 20 * int(sampleRate); i < 40*int(sampleRate); i++ {
		t := float64(i) / float64(sampleRate)
		// Simulate speech with varying frequencies
		samples[i] = 0.15 * (math.Sin(2*math.Pi*200*t) +
			0.5*math.Sin(2*math.Pi*400*t) +
			0.3*math.Sin(2*math.Pi*800*t))
	}

	// 40-60s: clipped/distorted audio
	for i := 40 * int(sampleRate); i < 60*int(sampleRate); i++ {
		t := float64(i) / float64(sampleRate)
		val := 1.5 * math.Sin(2*math.Pi*300*t)
		if val > 1 {
			val = 1
		} else if val < -1 {
			val = -1
		}
		samples[i] = val
	}

	config := DefaultAnalyzeConfig()
	config.TargetDuration = 15.0
	config.TopN = 3

	segments, err := FindBestSegments(samples, sampleRate, config)
	if err != nil {
		t.Fatalf("FindBestSegments failed: %v", err)
	}

	if len(segments) == 0 {
		t.Fatal("no segments returned")
	}

	// Best segment should be in the 20-40s range
	best := segments[0]
	if best.StartTime < 15 || best.EndTime > 45 {
		t.Errorf("best segment %v-%v not in expected range (20-40s)",
			best.StartTime, best.EndTime)
	}

	// Segments should be sorted by score descending
	for i := 1; i < len(segments); i++ {
		if segments[i].Score > segments[i-1].Score {
			t.Errorf("segments not sorted by score: %v > %v",
				segments[i].Score, segments[i-1].Score)
		}
	}
}
