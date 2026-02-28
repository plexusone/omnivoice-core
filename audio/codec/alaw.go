package codec

// AlawEncode encodes 16-bit linear PCM samples to 8-bit A-law.
//
// A-law (G.711 A-law) is the standard audio codec for telephony in
// Europe and most countries outside North America/Japan.
// It compresses 16-bit PCM to 8-bit samples.
func AlawEncode(pcm []int16) []byte {
	result := make([]byte, len(pcm))
	for i, sample := range pcm {
		result[i] = alawEncodeSample(sample)
	}
	return result
}

// AlawDecode decodes 8-bit A-law samples to 16-bit linear PCM.
func AlawDecode(alaw []byte) []int16 {
	result := make([]int16, len(alaw))
	for i, sample := range alaw {
		result[i] = alawDecodeSample(sample)
	}
	return result
}

// AlawEncodeBytes encodes raw PCM bytes (little-endian int16) to A-law.
// This is a convenience function for working with byte slices directly.
func AlawEncodeBytes(pcm []byte) []byte {
	if len(pcm)%2 != 0 {
		// Pad with zero if odd length
		pcm = append(pcm, 0)
	}

	result := make([]byte, len(pcm)/2)
	for i := 0; i < len(pcm); i += 2 {
		// Little-endian: low byte first
		sample := int16(pcm[i]) | int16(pcm[i+1])<<8
		result[i/2] = alawEncodeSample(sample)
	}
	return result
}

// AlawDecodeBytes decodes A-law to raw PCM bytes (little-endian int16).
// This is a convenience function for working with byte slices directly.
func AlawDecodeBytes(alaw []byte) []byte {
	result := make([]byte, len(alaw)*2)
	for i, sample := range alaw {
		decoded := alawDecodeSample(sample)
		// Little-endian: low byte first
		result[i*2] = uint8(decoded & 0xFF)          //nolint:gosec // G115: intentional low byte extraction
		result[i*2+1] = uint8((decoded >> 8) & 0xFF) //nolint:gosec // G115: intentional high byte extraction
	}
	return result
}

// alawEncodeSample encodes a single 16-bit PCM sample to 8-bit A-law.
//
// The A-law encoding formula is:
//
//	        A|x|/(1+ln(A))           for |x| < 1/A
//	F(x) = sgn(x) *
//	        (1+ln(A|x|))/(1+ln(A))   for 1/A <= |x| <= 1
//
// where A = 87.6 for standard telephony.
//
// This implementation uses the bit-manipulation method for efficiency.
func alawEncodeSample(sample int16) byte {
	// Determine sign
	var sign byte
	if sample >= 0 {
		sign = 0xD5 // Even bits inverted for positive
	} else {
		sign = 0x55 // Even bits inverted for negative
		if sample == -32768 {
			sample = 32767 // Handle int16 min edge case
		} else {
			sample = -sample
		}
	}

	var companded byte

	if sample < 256 {
		// Linear region
		companded = byte(sample >> 4)
	} else {
		// Find segment (exponent)
		var exponent byte
		shifted := sample >> 8
		for exponent = 1; exponent < 8; exponent++ {
			if shifted < 2 {
				break
			}
			shifted >>= 1
		}

		// Extract mantissa
		mantissa := byte((sample >> (exponent + 3)) & 0x0F)

		// Combine exponent and mantissa
		companded = (exponent << 4) | mantissa
	}

	// XOR with sign pattern (inverts even bits)
	return companded ^ sign
}

// alawDecodeSample decodes a single 8-bit A-law sample to 16-bit PCM.
func alawDecodeSample(alaw byte) int16 {
	// XOR to undo the even-bit inversion
	alaw ^= 0x55

	// Extract sign, exponent, and mantissa
	sign := alaw & 0x80
	exponent := (alaw >> 4) & 0x07
	mantissa := alaw & 0x0F

	var sample int16

	if exponent == 0 {
		// Linear region
		sample = int16(mantissa<<4) + 8
	} else {
		// Companded region
		sample = int16((int(mantissa)<<4 + 0x108) << (exponent - 1)) //nolint:gosec // G115: intentional codec conversion
	}

	// Apply sign (sign bit 1 = positive, 0 = negative in A-law)
	if sign == 0 {
		return -sample
	}
	return sample
}

// AlawSilence returns a byte representing silence in A-law encoding.
// This is useful for generating silence or padding.
func AlawSilence() byte {
	return 0xD5 // A-law encoding of 0 (silence)
}

// AlawSilenceBuffer returns a buffer of silence in A-law encoding.
func AlawSilenceBuffer(length int) []byte {
	result := make([]byte, length)
	for i := range result {
		result[i] = 0xD5
	}
	return result
}

// MulawToAlaw converts mu-law encoded audio to A-law encoding.
// This is useful when bridging between North American and European systems.
func MulawToAlaw(mulaw []byte) []byte {
	result := make([]byte, len(mulaw))
	for i, sample := range mulaw {
		// Decode mu-law to PCM, then encode to A-law
		pcm := mulawDecodeSample(sample)
		result[i] = alawEncodeSample(pcm)
	}
	return result
}

// AlawToMulaw converts A-law encoded audio to mu-law encoding.
// This is useful when bridging between European and North American systems.
func AlawToMulaw(alaw []byte) []byte {
	result := make([]byte, len(alaw))
	for i, sample := range alaw {
		// Decode A-law to PCM, then encode to mu-law
		pcm := alawDecodeSample(sample)
		result[i] = mulawEncodeSample(pcm)
	}
	return result
}
