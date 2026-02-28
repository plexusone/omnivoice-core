// Package codec provides audio codec implementations for telephony.
//
// This package implements G.711 codecs (mu-law and A-law) which are standard
// audio compression formats used in telephony systems worldwide.
//
//   - mu-law (μ-law): Used in North America and Japan
//   - A-law: Used in Europe and most other countries
//
// Both codecs compress 16-bit linear PCM samples to 8-bit companded samples,
// providing 2:1 compression while maintaining good voice quality.
package codec

// MulawEncode encodes 16-bit linear PCM samples to 8-bit mu-law.
//
// mu-law (G.711 μ-law) is the standard audio codec for telephony in
// North America and Japan. It compresses 16-bit PCM to 8-bit samples.
func MulawEncode(pcm []int16) []byte {
	result := make([]byte, len(pcm))
	for i, sample := range pcm {
		result[i] = mulawEncodeSample(sample)
	}
	return result
}

// MulawDecode decodes 8-bit mu-law samples to 16-bit linear PCM.
func MulawDecode(mulaw []byte) []int16 {
	result := make([]int16, len(mulaw))
	for i, sample := range mulaw {
		result[i] = mulawDecodeSample(sample)
	}
	return result
}

// MulawEncodeBytes encodes raw PCM bytes (little-endian int16) to mu-law.
// This is a convenience function for working with byte slices directly.
func MulawEncodeBytes(pcm []byte) []byte {
	if len(pcm)%2 != 0 {
		// Pad with zero if odd length
		pcm = append(pcm, 0)
	}

	result := make([]byte, len(pcm)/2)
	for i := 0; i < len(pcm); i += 2 {
		// Little-endian: low byte first
		sample := int16(pcm[i]) | int16(pcm[i+1])<<8
		result[i/2] = mulawEncodeSample(sample)
	}
	return result
}

// MulawDecodeBytes decodes mu-law to raw PCM bytes (little-endian int16).
// This is a convenience function for working with byte slices directly.
func MulawDecodeBytes(mulaw []byte) []byte {
	result := make([]byte, len(mulaw)*2)
	for i, sample := range mulaw {
		decoded := mulawDecodeSample(sample)
		// Little-endian: low byte first
		result[i*2] = uint8(decoded & 0xFF)          //nolint:gosec // G115: intentional low byte extraction
		result[i*2+1] = uint8((decoded >> 8) & 0xFF) //nolint:gosec // G115: intentional high byte extraction
	}
	return result
}

// mulawEncodeSample encodes a single 16-bit PCM sample to 8-bit mu-law.
//
// The mu-law encoding formula is:
//
//	F(x) = sgn(x) * ln(1 + μ|x|) / ln(1 + μ)
//
// where μ = 255 for standard telephony.
//
// This implementation uses the bit-manipulation method for efficiency.
func mulawEncodeSample(sample int16) byte {
	// Determine sign and get absolute value
	var sign byte
	if sample < 0 {
		sign = 0x80
		if sample == -32768 {
			sample = 32767 // Handle int16 min edge case
		} else {
			sample = -sample
		}
	}

	// Clip to 14-bit range (mu-law uses 14-bit input)
	if sample > 32635 {
		sample = 32635
	}

	// Add bias for rounding
	sample += 0x84

	// Find the segment (exponent)
	var exponent byte
	for i := 7; i >= 0; i-- {
		if sample >= (1 << uint(i+7)) { //nolint:gosec // G115: i is always in range [0,7]
			exponent = byte(i)
			break
		}
	}

	// Extract mantissa
	mantissa := byte((sample >> (exponent + 3)) & 0x0F)

	// Combine sign, exponent, and mantissa, then invert
	return ^(sign | (exponent << 4) | mantissa)
}

// mulawDecodeSample decodes a single 8-bit mu-law sample to 16-bit PCM.
func mulawDecodeSample(mulaw byte) int16 {
	// Invert the bits
	mulaw = ^mulaw

	// Extract sign, exponent, and mantissa
	sign := mulaw & 0x80
	exponent := (mulaw >> 4) & 0x07
	mantissa := mulaw & 0x0F

	// Reconstruct the linear value
	// Add 0.5 (in the form of 0x84 bias) and shift
	sample := int16((int(mantissa)<<3 + 0x84) << exponent) //nolint:gosec // G115: intentional codec conversion

	// Remove bias
	sample -= 0x84

	// Apply sign
	if sign != 0 {
		return -sample
	}
	return sample
}

// MulawSilence returns a byte representing silence in mu-law encoding.
// This is useful for generating silence or padding.
func MulawSilence() byte {
	return 0xFF // mu-law encoding of 0 (silence)
}

// MulawSilenceBuffer returns a buffer of silence in mu-law encoding.
func MulawSilenceBuffer(length int) []byte {
	result := make([]byte, length)
	for i := range result {
		result[i] = 0xFF
	}
	return result
}
