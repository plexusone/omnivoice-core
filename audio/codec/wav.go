package codec

import "encoding/binary"

// EncodeWAV wraps PCM samples in a WAV file container.
// The input should be 16-bit signed PCM samples.
// Returns a complete WAV file as bytes.
func EncodeWAV(samples []int16, sampleRate int, channels int) []byte {
	if channels == 0 {
		channels = 1
	}
	if sampleRate == 0 {
		sampleRate = 8000
	}

	bitsPerSample := 16
	bytesPerSample := bitsPerSample / 8
	dataSize := len(samples) * bytesPerSample
	fileSize := 44 + dataSize - 8 // 44 byte header, minus 8 for RIFF header

	// Build WAV file
	wav := make([]byte, 44+dataSize)

	// RIFF chunk
	copy(wav[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(fileSize)) //nolint:gosec // G115: bounded by sample count
	copy(wav[8:12], "WAVE")

	// fmt subchunk
	copy(wav[12:16], "fmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)                                          // Subchunk1Size (16 for PCM)
	binary.LittleEndian.PutUint16(wav[20:22], 1)                                           // AudioFormat (1 = PCM)
	binary.LittleEndian.PutUint16(wav[22:24], uint16(channels))                            //nolint:gosec // G115: channels is 1 or 2
	binary.LittleEndian.PutUint32(wav[24:28], uint32(sampleRate))                          //nolint:gosec // G115: sample rate is bounded
	binary.LittleEndian.PutUint32(wav[28:32], uint32(sampleRate*channels*bytesPerSample))  //nolint:gosec // G115: byte rate is bounded
	binary.LittleEndian.PutUint16(wav[32:34], uint16(channels*bytesPerSample))             //nolint:gosec // G115: block align is small
	binary.LittleEndian.PutUint16(wav[34:36], uint16(bitsPerSample))                       //nolint:gosec // G115: bits per sample is 16

	// data subchunk
	copy(wav[36:40], "data")
	binary.LittleEndian.PutUint32(wav[40:44], uint32(dataSize)) //nolint:gosec // G115: bounded by sample count

	// Write PCM data (little-endian)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(wav[44+i*2:], uint16(sample)) //nolint:gosec // G115: int16->uint16 is standard PCM conversion
	}

	return wav
}

// EncodeWAVBytes wraps raw PCM bytes in a WAV file container.
// The input should be little-endian 16-bit signed PCM bytes.
func EncodeWAVBytes(pcmData []byte, sampleRate int, channels int) []byte {
	samples := BytesToInt16(pcmData, false)
	return EncodeWAV(samples, sampleRate, channels)
}

// MulawToWAV converts μ-law encoded audio to a WAV file.
// The μ-law audio is assumed to be 8kHz mono.
func MulawToWAV(mulaw []byte) []byte {
	// Decode μ-law to linear PCM
	pcm := MulawDecode(mulaw)
	// Wrap in WAV container
	return EncodeWAV(pcm, 8000, 1)
}
