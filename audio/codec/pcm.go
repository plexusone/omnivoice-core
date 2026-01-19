package codec

import (
	"encoding/binary"
	"math"
)

// SampleRate represents common audio sample rates.
type SampleRate int

const (
	// SampleRate8kHz is the standard telephony sample rate.
	SampleRate8kHz SampleRate = 8000

	// SampleRate16kHz is common for speech recognition.
	SampleRate16kHz SampleRate = 16000

	// SampleRate22050Hz is a common intermediate rate.
	SampleRate22050Hz SampleRate = 22050

	// SampleRate44100Hz is CD quality audio.
	SampleRate44100Hz SampleRate = 44100

	// SampleRate48kHz is professional audio/video standard.
	SampleRate48kHz SampleRate = 48000
)

// PCMFormat describes the format of PCM audio data.
type PCMFormat struct {
	SampleRate SampleRate
	Channels   int  // 1 for mono, 2 for stereo
	BitDepth   int  // 8, 16, 24, or 32
	BigEndian  bool // true for big-endian, false for little-endian
}

// TelephonyFormat returns the standard telephony PCM format.
// This is 8kHz, mono, 16-bit, little-endian.
func TelephonyFormat() PCMFormat {
	return PCMFormat{
		SampleRate: SampleRate8kHz,
		Channels:   1,
		BitDepth:   16,
		BigEndian:  false,
	}
}

// Int16ToBytes converts int16 samples to bytes in the specified endianness.
func Int16ToBytes(samples []int16, bigEndian bool) []byte {
	result := make([]byte, len(samples)*2)
	for i, sample := range samples {
		if bigEndian {
			binary.BigEndian.PutUint16(result[i*2:], uint16(sample)) //nolint:gosec // G115: int16->uint16 is standard PCM conversion
		} else {
			binary.LittleEndian.PutUint16(result[i*2:], uint16(sample)) //nolint:gosec // G115: int16->uint16 is standard PCM conversion
		}
	}
	return result
}

// BytesToInt16 converts bytes to int16 samples with the specified endianness.
func BytesToInt16(data []byte, bigEndian bool) []int16 {
	if len(data)%2 != 0 {
		// Pad with zero if odd length
		data = append(data, 0)
	}

	result := make([]int16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		if bigEndian {
			result[i/2] = int16(binary.BigEndian.Uint16(data[i:])) //nolint:gosec // G115: uint16->int16 is standard PCM conversion
		} else {
			result[i/2] = int16(binary.LittleEndian.Uint16(data[i:])) //nolint:gosec // G115: uint16->int16 is standard PCM conversion
		}
	}
	return result
}

// Float32ToInt16 converts float32 samples (range -1.0 to 1.0) to int16.
func Float32ToInt16(samples []float32) []int16 {
	result := make([]int16, len(samples))
	for i, sample := range samples {
		// Clamp to valid range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		result[i] = int16(sample * 32767)
	}
	return result
}

// Int16ToFloat32 converts int16 samples to float32 (range -1.0 to 1.0).
func Int16ToFloat32(samples []int16) []float32 {
	result := make([]float32, len(samples))
	for i, sample := range samples {
		result[i] = float32(sample) / 32768.0
	}
	return result
}

// Float64ToInt16 converts float64 samples (range -1.0 to 1.0) to int16.
func Float64ToInt16(samples []float64) []int16 {
	result := make([]int16, len(samples))
	for i, sample := range samples {
		// Clamp to valid range
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		result[i] = int16(sample * 32767)
	}
	return result
}

// Int16ToFloat64 converts int16 samples to float64 (range -1.0 to 1.0).
func Int16ToFloat64(samples []int16) []float64 {
	result := make([]float64, len(samples))
	for i, sample := range samples {
		result[i] = float64(sample) / 32768.0
	}
	return result
}

// MonoToStereo converts mono audio to stereo by duplicating samples.
func MonoToStereo(mono []int16) []int16 {
	stereo := make([]int16, len(mono)*2)
	for i, sample := range mono {
		stereo[i*2] = sample   // Left channel
		stereo[i*2+1] = sample // Right channel
	}
	return stereo
}

// StereoToMono converts stereo audio to mono by averaging channels.
func StereoToMono(stereo []int16) []int16 {
	if len(stereo)%2 != 0 {
		// Handle odd length by padding
		stereo = append(stereo, 0)
	}

	mono := make([]int16, len(stereo)/2)
	for i := 0; i < len(stereo); i += 2 {
		// Average left and right channels
		left := int32(stereo[i])
		right := int32(stereo[i+1])
		mono[i/2] = int16((left + right) / 2) //nolint:gosec // G115: average of two int16 values fits in int16
	}
	return mono
}

// Resample performs simple linear interpolation resampling.
// For production use, consider using a proper resampling library.
func Resample(samples []int16, fromRate, toRate SampleRate) []int16 {
	if fromRate == toRate {
		return samples
	}

	ratio := float64(toRate) / float64(fromRate)
	newLength := int(float64(len(samples)) * ratio)
	result := make([]int16, newLength)

	for i := range newLength {
		srcIndex := float64(i) / ratio
		srcIndexInt := int(srcIndex)
		frac := srcIndex - float64(srcIndexInt)

		if srcIndexInt >= len(samples)-1 {
			result[i] = samples[len(samples)-1]
		} else {
			// Linear interpolation
			sample1 := float64(samples[srcIndexInt])
			sample2 := float64(samples[srcIndexInt+1])
			result[i] = int16(sample1 + frac*(sample2-sample1))
		}
	}

	return result
}

// ResampleBytes resamples raw PCM bytes (little-endian int16).
func ResampleBytes(data []byte, fromRate, toRate SampleRate) []byte {
	samples := BytesToInt16(data, false)
	resampled := Resample(samples, fromRate, toRate)
	return Int16ToBytes(resampled, false)
}

// NormalizePCM normalizes PCM samples to use the full dynamic range.
// This finds the peak amplitude and scales all samples proportionally.
func NormalizePCM(samples []int16) []int16 {
	if len(samples) == 0 {
		return samples
	}

	// Find peak amplitude
	var maxAmp int16
	for _, sample := range samples {
		if sample < 0 {
			sample = -sample
		}
		if sample > maxAmp {
			maxAmp = sample
		}
	}

	if maxAmp == 0 {
		return samples
	}

	// Calculate scaling factor
	scale := float64(32767) / float64(maxAmp)

	// Apply scaling
	result := make([]int16, len(samples))
	for i, sample := range samples {
		result[i] = int16(float64(sample) * scale)
	}

	return result
}

// CalculateRMS calculates the Root Mean Square of PCM samples.
// This is a measure of audio loudness.
func CalculateRMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample) * float64(sample)
	}

	return math.Sqrt(sum / float64(len(samples)))
}

// CalculateDBFS calculates the level in dB relative to full scale.
// 0 dBFS is the maximum possible level, negative values indicate quieter audio.
func CalculateDBFS(samples []int16) float64 {
	rms := CalculateRMS(samples)
	if rms == 0 {
		return -96.0 // Effectively silence
	}
	return 20 * math.Log10(rms/32768.0)
}

// GenerateSilence generates a buffer of PCM silence (zeros).
func GenerateSilence(numSamples int) []int16 {
	return make([]int16, numSamples)
}

// GenerateSilenceBytes generates a buffer of PCM silence as bytes.
func GenerateSilenceBytes(numSamples int) []byte {
	return make([]byte, numSamples*2)
}

// GenerateSineWave generates a sine wave tone.
// Useful for testing audio pipelines.
func GenerateSineWave(frequency float64, sampleRate SampleRate, duration float64, amplitude int16) []int16 {
	numSamples := int(float64(sampleRate) * duration)
	samples := make([]int16, numSamples)

	for i := range numSamples {
		t := float64(i) / float64(sampleRate)
		samples[i] = int16(float64(amplitude) * math.Sin(2*math.Pi*frequency*t))
	}

	return samples
}
