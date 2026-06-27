// Package audio provides audio analysis utilities.
package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
)

// SegmentScore represents a scored segment of audio.
type SegmentScore struct {
	StartSample int
	EndSample   int
	StartTime   float64
	EndTime     float64
	Score       float64
	Metrics     SegmentMetrics
}

// SegmentMetrics contains quality metrics for a segment.
type SegmentMetrics struct {
	// MeanEnergy is the average RMS energy (higher = louder speech)
	MeanEnergy float64

	// EnergyVariance measures consistency (lower = more consistent)
	EnergyVariance float64

	// SpeechRatio is the fraction of frames with speech activity
	SpeechRatio float64

	// SilenceGaps is the number of silence gaps > 0.5s
	SilenceGaps int

	// MaxSilenceGap is the longest silence gap in seconds
	MaxSilenceGap float64

	// PeakClipping is the fraction of samples near clipping
	PeakClipping float64
}

// WAVHeader represents a WAV file header.
type WAVHeader struct {
	SampleRate    uint32
	NumChannels   uint16
	BitsPerSample uint16
	DataSize      uint32
}

// AnalyzeConfig configures the segment analysis.
type AnalyzeConfig struct {
	// TargetDuration is the desired segment length in seconds
	TargetDuration float64

	// StepSize is the sliding window step in seconds
	StepSize float64

	// FrameSize is the analysis frame size in seconds
	FrameSize float64

	// SilenceThreshold is the RMS threshold for silence detection (0-1)
	SilenceThreshold float64

	// TopN returns the top N segments
	TopN int
}

// DefaultAnalyzeConfig returns sensible defaults.
func DefaultAnalyzeConfig() AnalyzeConfig {
	return AnalyzeConfig{
		TargetDuration:   15.0,
		StepSize:         1.0,
		FrameSize:        0.025, // 25ms frames
		SilenceThreshold: 0.02,
		TopN:             5,
	}
}

// ReadWAV reads a WAV file and returns the header and samples as float64 (-1 to 1).
func ReadWAV(path string) (*WAVHeader, []float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Read RIFF header
	var riffHeader [12]byte
	if _, err := io.ReadFull(f, riffHeader[:]); err != nil {
		return nil, nil, fmt.Errorf("failed to read RIFF header: %w", err)
	}

	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return nil, nil, fmt.Errorf("not a valid WAV file")
	}

	var header WAVHeader

	// Parse chunks
	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(f, chunkHeader[:]); err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("failed to read chunk header: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			var fmtData [16]byte
			if _, err := io.ReadFull(f, fmtData[:]); err != nil {
				return nil, nil, fmt.Errorf("failed to read fmt chunk: %w", err)
			}

			audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
			if audioFormat != 1 {
				return nil, nil, fmt.Errorf("unsupported audio format: %d (only PCM supported)", audioFormat)
			}

			header.NumChannels = binary.LittleEndian.Uint16(fmtData[2:4])
			header.SampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
			header.BitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])

			// Skip any extra fmt bytes
			if chunkSize > 16 {
				if _, err := f.Seek(int64(chunkSize-16), io.SeekCurrent); err != nil {
					return nil, nil, err
				}
			}

		case "data":
			header.DataSize = chunkSize

			// Read audio data
			data := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, data); err != nil {
				return nil, nil, fmt.Errorf("failed to read audio data: %w", err)
			}

			samples, err := bytesToSamples(data, header.BitsPerSample, header.NumChannels)
			if err != nil {
				return nil, nil, err
			}

			return &header, samples, nil

		default:
			// Skip unknown chunks
			if _, err := f.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return nil, nil, err
			}
		}
	}

	return nil, nil, fmt.Errorf("no data chunk found")
}

// bytesToSamples converts raw bytes to float64 samples, mixing to mono.
func bytesToSamples(data []byte, bitsPerSample, numChannels uint16) ([]float64, error) {
	bytesPerSample := int(bitsPerSample / 8)
	bytesPerFrame := bytesPerSample * int(numChannels)
	numFrames := len(data) / bytesPerFrame

	samples := make([]float64, numFrames)

	for i := 0; i < numFrames; i++ {
		offset := i * bytesPerFrame
		var sum float64

		for ch := 0; ch < int(numChannels); ch++ {
			chOffset := offset + ch*bytesPerSample
			var sample float64

			switch bitsPerSample {
			case 8:
				// 8-bit is unsigned
				sample = (float64(data[chOffset]) - 128) / 128
			case 16:
				// 16-bit is signed little-endian
				val := int16(binary.LittleEndian.Uint16(data[chOffset : chOffset+2]))
				sample = float64(val) / 32768
			case 24:
				// 24-bit is signed little-endian
				val := int32(data[chOffset]) | int32(data[chOffset+1])<<8 | int32(data[chOffset+2])<<16
				if val&0x800000 != 0 {
					val |= ^0xFFFFFF // Sign extend
				}
				sample = float64(val) / 8388608
			case 32:
				// 32-bit is signed little-endian
				val := int32(binary.LittleEndian.Uint32(data[chOffset : chOffset+4]))
				sample = float64(val) / 2147483648
			default:
				return nil, fmt.Errorf("unsupported bits per sample: %d", bitsPerSample)
			}

			sum += sample
		}

		// Average channels to mono
		samples[i] = sum / float64(numChannels)
	}

	return samples, nil
}

// FindBestSegments analyzes audio and returns the best segments for voice cloning.
func FindBestSegments(samples []float64, sampleRate uint32, config AnalyzeConfig) ([]SegmentScore, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("no audio samples")
	}

	totalDuration := float64(len(samples)) / float64(sampleRate)
	if totalDuration < config.TargetDuration {
		return nil, fmt.Errorf("audio too short: %.1fs < %.1fs target", totalDuration, config.TargetDuration)
	}

	targetSamples := int(config.TargetDuration * float64(sampleRate))
	stepSamples := int(config.StepSize * float64(sampleRate))
	frameSamples := int(config.FrameSize * float64(sampleRate))

	var segments []SegmentScore

	// Slide window across audio
	for start := 0; start+targetSamples <= len(samples); start += stepSamples {
		end := start + targetSamples
		segment := samples[start:end]

		metrics := analyzeSegment(segment, sampleRate, frameSamples, config.SilenceThreshold)
		score := scoreMetrics(metrics)

		segments = append(segments, SegmentScore{
			StartSample: start,
			EndSample:   end,
			StartTime:   float64(start) / float64(sampleRate),
			EndTime:     float64(end) / float64(sampleRate),
			Score:       score,
			Metrics:     metrics,
		})
	}

	// Sort by score descending
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].Score > segments[j].Score
	})

	// Return top N
	if len(segments) > config.TopN {
		segments = segments[:config.TopN]
	}

	return segments, nil
}

// analyzeSegment computes quality metrics for a segment.
func analyzeSegment(samples []float64, sampleRate uint32, frameSamples int, silenceThreshold float64) SegmentMetrics {
	var metrics SegmentMetrics

	// Compute frame-level RMS energies
	numFrames := len(samples) / frameSamples
	if numFrames == 0 {
		return metrics
	}

	energies := make([]float64, numFrames)
	speechFrames := 0
	peakSamples := 0

	for i := 0; i < numFrames; i++ {
		start := i * frameSamples
		end := start + frameSamples
		if end > len(samples) {
			end = len(samples)
		}

		frame := samples[start:end]
		rms := computeRMS(frame)
		energies[i] = rms

		if rms > silenceThreshold {
			speechFrames++
		}

		// Check for clipping
		for _, s := range frame {
			if s > 0.99 || s < -0.99 {
				peakSamples++
			}
		}
	}

	// Mean energy
	var sumEnergy float64
	for _, e := range energies {
		sumEnergy += e
	}
	metrics.MeanEnergy = sumEnergy / float64(numFrames)

	// Energy variance
	var sumVariance float64
	for _, e := range energies {
		diff := e - metrics.MeanEnergy
		sumVariance += diff * diff
	}
	metrics.EnergyVariance = sumVariance / float64(numFrames)

	// Speech ratio
	metrics.SpeechRatio = float64(speechFrames) / float64(numFrames)

	// Peak clipping ratio
	metrics.PeakClipping = float64(peakSamples) / float64(len(samples))

	// Analyze silence gaps
	metrics.SilenceGaps, metrics.MaxSilenceGap = analyzeSilenceGaps(energies, silenceThreshold, float64(frameSamples)/float64(sampleRate))

	return metrics
}

// computeRMS computes root mean square of samples.
func computeRMS(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	for _, s := range samples {
		sum += s * s
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// analyzeSilenceGaps finds silence gaps in frame energies.
func analyzeSilenceGaps(energies []float64, threshold, frameDuration float64) (int, float64) {
	gaps := 0
	maxGap := 0.0
	currentGap := 0.0
	minGapDuration := 0.5 // Only count gaps > 0.5s

	for _, e := range energies {
		if e < threshold {
			currentGap += frameDuration
		} else {
			if currentGap > minGapDuration {
				gaps++
				if currentGap > maxGap {
					maxGap = currentGap
				}
			}
			currentGap = 0
		}
	}

	// Check final gap
	if currentGap > minGapDuration {
		gaps++
		if currentGap > maxGap {
			maxGap = currentGap
		}
	}

	return gaps, maxGap
}

// scoreMetrics computes an overall quality score from metrics.
// Higher score = better segment for voice cloning.
func scoreMetrics(m SegmentMetrics) float64 {
	score := 0.0

	// Prefer high speech ratio (0.6-0.9 is ideal)
	if m.SpeechRatio >= 0.6 && m.SpeechRatio <= 0.9 {
		score += 30
	} else if m.SpeechRatio >= 0.5 && m.SpeechRatio <= 0.95 {
		score += 20
	} else if m.SpeechRatio >= 0.4 {
		score += 10
	}

	// Prefer moderate energy (not too quiet, not too loud)
	if m.MeanEnergy >= 0.05 && m.MeanEnergy <= 0.3 {
		score += 25
	} else if m.MeanEnergy >= 0.03 && m.MeanEnergy <= 0.4 {
		score += 15
	} else if m.MeanEnergy >= 0.02 {
		score += 5
	}

	// Prefer consistent energy (low variance)
	normalizedVariance := m.EnergyVariance / (m.MeanEnergy + 0.001)
	if normalizedVariance < 0.5 {
		score += 20
	} else if normalizedVariance < 1.0 {
		score += 10
	}

	// Penalize silence gaps
	score -= float64(m.SilenceGaps) * 5
	if m.MaxSilenceGap > 1.0 {
		score -= 10
	}

	// Penalize clipping
	if m.PeakClipping > 0.01 {
		score -= 20
	} else if m.PeakClipping > 0.001 {
		score -= 5
	}

	return score
}

// ExtractSegment extracts a segment from samples.
func ExtractSegment(samples []float64, startSample, endSample int) []float64 {
	if startSample < 0 {
		startSample = 0
	}
	if endSample > len(samples) {
		endSample = len(samples)
	}
	result := make([]float64, endSample-startSample)
	copy(result, samples[startSample:endSample])
	return result
}

// WriteWAV writes samples to a WAV file.
func WriteWAV(path string, samples []float64, sampleRate uint32) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	numSamples := len(samples)
	dataSize := numSamples * 2 // 16-bit = 2 bytes per sample
	fileSize := 36 + dataSize

	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(fileSize))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))        // Chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))         // Audio format (PCM)
	binary.Write(f, binary.LittleEndian, uint16(1))         // Num channels (mono)
	binary.Write(f, binary.LittleEndian, sampleRate)        // Sample rate
	binary.Write(f, binary.LittleEndian, sampleRate*2)      // Byte rate
	binary.Write(f, binary.LittleEndian, uint16(2))         // Block align
	binary.Write(f, binary.LittleEndian, uint16(16))        // Bits per sample

	// data chunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	// Write samples as 16-bit signed integers
	for _, s := range samples {
		// Clamp to [-1, 1]
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		val := int16(s * 32767)
		binary.Write(f, binary.LittleEndian, val)
	}

	return nil
}
