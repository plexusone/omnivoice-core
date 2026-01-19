package codec

import (
	"math"
	"testing"
)

// codecTestCase defines a test case for codec round-trip tests.
type codecTestCase struct {
	name   string
	sample int16
}

// standardCodecTestCases returns the standard test cases for codec testing.
func standardCodecTestCases() []codecTestCase {
	return []codecTestCase{
		{"zero", 0},
		{"positive small", 100},
		{"positive large", 16000},
		{"max positive", 32767},
		{"negative small", -100},
		{"negative large", -16000},
		{"min negative", -32768},
	}
}

// testCodecRoundTrip is a helper that tests codec encode/decode round-trips.
func testCodecRoundTrip(t *testing.T, name string, encode func([]int16) []byte, decode func([]byte) []int16) {
	t.Helper()
	for _, tc := range standardCodecTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			input := []int16{tc.sample}
			encoded := encode(input)
			if len(encoded) != 1 {
				t.Fatalf("expected 1 byte, got %d", len(encoded))
			}

			decoded := decode(encoded)
			if len(decoded) != 1 {
				t.Fatalf("expected 1 sample, got %d", len(decoded))
			}

			// Companding codecs are lossy; check decoded value is within tolerance.
			// Tolerance depends on amplitude; larger values have more quantization error.
			tolerance := int16(500)
			if tc.sample > 1000 || tc.sample < -1000 {
				tolerance = int16(float64(abs(int(tc.sample))) * 0.1) // 10% tolerance
			}

			diff := abs(int(decoded[0]) - int(tc.sample))
			if diff > int(tolerance) {
				t.Errorf("%s: sample %d decoded to %d, diff %d exceeds tolerance %d",
					name, tc.sample, decoded[0], diff, tolerance)
			}
		})
	}
}

// TestMulawEncodeDecode tests mu-law round-trip encoding/decoding.
func TestMulawEncodeDecode(t *testing.T) {
	testCodecRoundTrip(t, "mu-law", MulawEncode, MulawDecode)
}

// TestMulawEncodeBytes tests byte-level encoding.
func TestMulawEncodeBytes(t *testing.T) {
	// Create PCM bytes (little-endian int16)
	pcm := []byte{0x00, 0x10} // 0x1000 = 4096
	encoded := MulawEncodeBytes(pcm)
	if len(encoded) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(encoded))
	}

	// Decode back
	decoded := MulawDecodeBytes(encoded)
	if len(decoded) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(decoded))
	}
}

// TestMulawSilence tests silence generation.
func TestMulawSilence(t *testing.T) {
	silence := MulawSilence()
	if silence != 0xFF {
		t.Errorf("expected 0xFF for silence, got 0x%02X", silence)
	}

	buffer := MulawSilenceBuffer(100)
	if len(buffer) != 100 {
		t.Fatalf("expected 100 bytes, got %d", len(buffer))
	}
	for i, b := range buffer {
		if b != 0xFF {
			t.Errorf("byte %d: expected 0xFF, got 0x%02X", i, b)
			break
		}
	}
}

// TestAlawEncodeDecode tests A-law round-trip encoding/decoding.
func TestAlawEncodeDecode(t *testing.T) {
	testCodecRoundTrip(t, "A-law", AlawEncode, AlawDecode)
}

// TestAlawSilence tests A-law silence generation.
func TestAlawSilence(t *testing.T) {
	silence := AlawSilence()
	if silence != 0xD5 {
		t.Errorf("expected 0xD5 for silence, got 0x%02X", silence)
	}

	buffer := AlawSilenceBuffer(100)
	if len(buffer) != 100 {
		t.Fatalf("expected 100 bytes, got %d", len(buffer))
	}
	for i, b := range buffer {
		if b != 0xD5 {
			t.Errorf("byte %d: expected 0xD5, got 0x%02X", i, b)
			break
		}
	}
}

// TestMulawToAlaw tests transcoding between mu-law and A-law.
func TestMulawToAlaw(t *testing.T) {
	// Create a test signal
	samples := []int16{0, 1000, -1000, 16000, -16000}
	mulaw := MulawEncode(samples)
	alaw := MulawToAlaw(mulaw)

	if len(alaw) != len(mulaw) {
		t.Fatalf("expected %d bytes, got %d", len(mulaw), len(alaw))
	}

	// Convert back and compare
	mulawBack := AlawToMulaw(alaw)
	if len(mulawBack) != len(mulaw) {
		t.Fatalf("expected %d bytes, got %d", len(mulaw), len(mulawBack))
	}

	// Decode both and compare (allowing for transcoding losses)
	originalDecoded := MulawDecode(mulaw)
	roundtripDecoded := MulawDecode(mulawBack)

	for i := range originalDecoded {
		diff := abs(int(originalDecoded[i]) - int(roundtripDecoded[i]))
		// Allow larger tolerance for transcoding
		tolerance := 1000
		if originalDecoded[i] > 1000 || originalDecoded[i] < -1000 {
			tolerance = int(float64(abs(int(originalDecoded[i]))) * 0.2)
		}
		if diff > tolerance {
			t.Errorf("sample %d: original %d vs roundtrip %d, diff %d",
				i, originalDecoded[i], roundtripDecoded[i], diff)
		}
	}
}

// TestInt16ToBytes tests byte conversion.
func TestInt16ToBytes(t *testing.T) {
	samples := []int16{0x0102, 0x0304}

	// Little-endian
	le := Int16ToBytes(samples, false)
	expected := []byte{0x02, 0x01, 0x04, 0x03}
	for i, b := range le {
		if b != expected[i] {
			t.Errorf("LE byte %d: expected 0x%02X, got 0x%02X", i, expected[i], b)
		}
	}

	// Big-endian
	be := Int16ToBytes(samples, true)
	expected = []byte{0x01, 0x02, 0x03, 0x04}
	for i, b := range be {
		if b != expected[i] {
			t.Errorf("BE byte %d: expected 0x%02X, got 0x%02X", i, expected[i], b)
		}
	}
}

// TestBytesToInt16 tests byte to sample conversion.
func TestBytesToInt16(t *testing.T) {
	data := []byte{0x02, 0x01, 0x04, 0x03}
	samples := BytesToInt16(data, false) // Little-endian
	expected := []int16{0x0102, 0x0304}

	for i, s := range samples {
		if s != expected[i] {
			t.Errorf("sample %d: expected %d, got %d", i, expected[i], s)
		}
	}

	// Odd length input
	oddData := []byte{0x02, 0x01, 0x04}
	oddSamples := BytesToInt16(oddData, false)
	if len(oddSamples) != 2 {
		t.Errorf("expected 2 samples for odd input, got %d", len(oddSamples))
	}
}

// TestFloat32Conversion tests float32 <-> int16 conversion.
func TestFloat32Conversion(t *testing.T) {
	samples := []int16{0, 16384, -16384, 32767, -32768}
	floats := Int16ToFloat32(samples)

	// Check ranges
	for i, f := range floats {
		if f < -1.0 || f > 1.0 {
			t.Errorf("sample %d: float %f out of range [-1, 1]", i, f)
		}
	}

	// Convert back
	back := Float32ToInt16(floats)
	for i, s := range back {
		// Allow small rounding error
		diff := abs(int(s) - int(samples[i]))
		if diff > 1 {
			t.Errorf("sample %d: expected %d, got %d", i, samples[i], s)
		}
	}

	// Test clamping
	outOfRange := []float32{2.0, -2.0}
	clamped := Float32ToInt16(outOfRange)
	if clamped[0] != 32767 {
		t.Errorf("expected 32767 for 2.0, got %d", clamped[0])
	}
	if clamped[1] != -32767 {
		t.Errorf("expected -32767 for -2.0, got %d", clamped[1])
	}
}

// TestFloat64Conversion tests float64 <-> int16 conversion.
func TestFloat64Conversion(t *testing.T) {
	samples := []int16{0, 16384, -16384, 32767, -32768}
	floats := Int16ToFloat64(samples)

	// Check ranges
	for i, f := range floats {
		if f < -1.0 || f > 1.0 {
			t.Errorf("sample %d: float %f out of range [-1, 1]", i, f)
		}
	}

	// Convert back
	back := Float64ToInt16(floats)
	for i, s := range back {
		diff := abs(int(s) - int(samples[i]))
		if diff > 1 {
			t.Errorf("sample %d: expected %d, got %d", i, samples[i], s)
		}
	}
}

// TestMonoStereoConversion tests mono <-> stereo conversion.
func TestMonoStereoConversion(t *testing.T) {
	mono := []int16{100, 200, 300}
	stereo := MonoToStereo(mono)

	if len(stereo) != 6 {
		t.Fatalf("expected 6 samples, got %d", len(stereo))
	}

	// Check interleaving
	for i, m := range mono {
		if stereo[i*2] != m || stereo[i*2+1] != m {
			t.Errorf("sample %d: expected (%d, %d), got (%d, %d)",
				i, m, m, stereo[i*2], stereo[i*2+1])
		}
	}

	// Convert back
	monoBack := StereoToMono(stereo)
	if len(monoBack) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(monoBack))
	}
	for i, m := range monoBack {
		if m != mono[i] {
			t.Errorf("sample %d: expected %d, got %d", i, mono[i], m)
		}
	}
}

// TestResample tests basic resampling.
func TestResample(t *testing.T) {
	// Same rate should return original
	samples := []int16{100, 200, 300, 400}
	same := Resample(samples, SampleRate8kHz, SampleRate8kHz)
	if len(same) != len(samples) {
		t.Errorf("same rate: expected %d samples, got %d", len(samples), len(same))
	}

	// Upsample 8kHz to 16kHz (should double length)
	upsampled := Resample(samples, SampleRate8kHz, SampleRate16kHz)
	expectedLen := len(samples) * 2
	if len(upsampled) != expectedLen {
		t.Errorf("upsample: expected %d samples, got %d", expectedLen, len(upsampled))
	}

	// Downsample 16kHz to 8kHz (should halve length)
	longSamples := make([]int16, 100)
	for i := range longSamples {
		longSamples[i] = int16(i * 100) //nolint:gosec // G115: i*100 is always < 10000, fits in int16
	}
	downsampled := Resample(longSamples, SampleRate16kHz, SampleRate8kHz)
	if len(downsampled) != 50 {
		t.Errorf("downsample: expected 50 samples, got %d", len(downsampled))
	}
}

// TestNormalizePCM tests audio normalization.
func TestNormalizePCM(t *testing.T) {
	// Empty input
	empty := NormalizePCM(nil)
	if len(empty) != 0 {
		t.Error("expected empty output for nil input")
	}

	// All zeros
	zeros := []int16{0, 0, 0}
	normalized := NormalizePCM(zeros)
	for i, s := range normalized {
		if s != 0 {
			t.Errorf("sample %d: expected 0, got %d", i, s)
		}
	}

	// Half amplitude should be normalized to full
	half := []int16{16383, -16383, 8000}
	normalized = NormalizePCM(half)
	// Peak should now be close to 32767
	var maxAmp int16
	for _, s := range normalized {
		if s < 0 {
			s = -s
		}
		if s > maxAmp {
			maxAmp = s
		}
	}
	if maxAmp < 32700 {
		t.Errorf("expected peak near 32767, got %d", maxAmp)
	}
}

// TestCalculateRMS tests RMS calculation.
func TestCalculateRMS(t *testing.T) {
	// Empty
	if rms := CalculateRMS(nil); rms != 0 {
		t.Errorf("expected 0 RMS for nil, got %f", rms)
	}

	// Silence
	silence := []int16{0, 0, 0}
	if rms := CalculateRMS(silence); rms != 0 {
		t.Errorf("expected 0 RMS for silence, got %f", rms)
	}

	// Full scale
	fullScale := []int16{32767, 32767, 32767}
	rms := CalculateRMS(fullScale)
	if rms < 32700 {
		t.Errorf("expected RMS near 32767, got %f", rms)
	}
}

// TestCalculateDBFS tests dBFS calculation.
func TestCalculateDBFS(t *testing.T) {
	// Silence should return very low dB
	silence := []int16{0, 0, 0}
	dbfs := CalculateDBFS(silence)
	if dbfs != -96.0 {
		t.Errorf("expected -96 dBFS for silence, got %f", dbfs)
	}

	// Full scale should be close to 0 dBFS
	fullScale := []int16{32767, 32767, 32767}
	dbfs = CalculateDBFS(fullScale)
	if dbfs < -1.0 || dbfs > 0 {
		t.Errorf("expected dBFS near 0 for full scale, got %f", dbfs)
	}
}

// TestGenerateSilence tests silence generation.
func TestGenerateSilence(t *testing.T) {
	samples := GenerateSilence(100)
	if len(samples) != 100 {
		t.Fatalf("expected 100 samples, got %d", len(samples))
	}
	for i, s := range samples {
		if s != 0 {
			t.Errorf("sample %d: expected 0, got %d", i, s)
		}
	}

	bytes := GenerateSilenceBytes(100)
	if len(bytes) != 200 {
		t.Fatalf("expected 200 bytes, got %d", len(bytes))
	}
}

// TestGenerateSineWave tests sine wave generation.
func TestGenerateSineWave(t *testing.T) {
	duration := 0.01             // 10ms
	sampleRate := SampleRate8kHz // 8000 Hz
	frequency := 440.0           // A4
	amplitude := int16(16383)    // Half scale
	samples := GenerateSineWave(frequency, sampleRate, duration, amplitude)

	expectedLen := int(float64(sampleRate) * duration)
	if len(samples) != expectedLen {
		t.Fatalf("expected %d samples, got %d", expectedLen, len(samples))
	}

	// Check that samples are within amplitude range
	for i, s := range samples {
		if s > amplitude || s < -amplitude {
			t.Errorf("sample %d: %d exceeds amplitude %d", i, s, amplitude)
		}
	}

	// Verify it's not all zeros
	hasNonZero := false
	for _, s := range samples {
		if s != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("sine wave should have non-zero samples")
	}
}

// TestTelephonyFormat tests the standard telephony format.
func TestTelephonyFormat(t *testing.T) {
	format := TelephonyFormat()
	if format.SampleRate != SampleRate8kHz {
		t.Errorf("expected 8kHz, got %d", format.SampleRate)
	}
	if format.Channels != 1 {
		t.Errorf("expected mono (1), got %d", format.Channels)
	}
	if format.BitDepth != 16 {
		t.Errorf("expected 16-bit, got %d", format.BitDepth)
	}
	if format.BigEndian {
		t.Error("expected little-endian")
	}
}

// abs returns the absolute value of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// BenchmarkMulawEncode benchmarks mu-law encoding.
func BenchmarkMulawEncode(b *testing.B) {
	// Simulate 20ms of audio at 8kHz
	samples := make([]int16, 160)
	for i := range samples {
		samples[i] = int16(math.Sin(float64(i)*0.1) * 16000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MulawEncode(samples)
	}
}

// BenchmarkMulawDecode benchmarks mu-law decoding.
func BenchmarkMulawDecode(b *testing.B) {
	samples := make([]int16, 160)
	for i := range samples {
		samples[i] = int16(math.Sin(float64(i)*0.1) * 16000)
	}
	encoded := MulawEncode(samples)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MulawDecode(encoded)
	}
}

// BenchmarkAlawEncode benchmarks A-law encoding.
func BenchmarkAlawEncode(b *testing.B) {
	samples := make([]int16, 160)
	for i := range samples {
		samples[i] = int16(math.Sin(float64(i)*0.1) * 16000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AlawEncode(samples)
	}
}

// BenchmarkResample benchmarks resampling.
func BenchmarkResample(b *testing.B) {
	// 20ms at 8kHz
	samples := make([]int16, 160)
	for i := range samples {
		samples[i] = int16(math.Sin(float64(i)*0.1) * 16000)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Resample(samples, SampleRate8kHz, SampleRate16kHz)
	}
}
