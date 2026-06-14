package converter

import (
	"errors"

	"github.com/plexusone/omnivoice-core/audio/codec"
	"github.com/plexusone/omnivoice-core/audio/format"
)

// Errors returned by the converter.
var (
	ErrUnsupportedEncoding   = errors.New("unsupported audio encoding")
	ErrUnsupportedSampleRate = errors.New("unsupported sample rate")
	ErrInvalidAudioData      = errors.New("invalid audio data")
)

// Converter converts audio between telephony and realtime provider formats.
type Converter struct{}

// New creates a new audio converter.
func New() *Converter {
	return &Converter{}
}

// Convert converts audio from one format to another.
// Handles both codec conversion (mulaw ↔ PCM16) and sample rate conversion.
func (c *Converter) Convert(audio []byte, from, to format.AudioFormat) ([]byte, error) {
	if len(audio) == 0 {
		return audio, nil
	}

	// Step 1: Decode to PCM16 at source sample rate
	pcm, err := c.decode(audio, from)
	if err != nil {
		return nil, err
	}

	// Step 2: Resample if needed
	if from.SampleRate != to.SampleRate {
		pcm = codec.Resample(pcm, codec.SampleRate(from.SampleRate), codec.SampleRate(to.SampleRate))
	}

	// Step 3: Handle channel conversion if needed
	if from.Channels != to.Channels {
		if from.Channels == 1 && to.Channels == 2 {
			pcm = codec.MonoToStereo(pcm)
		} else if from.Channels == 2 && to.Channels == 1 {
			pcm = codec.StereoToMono(pcm)
		}
	}

	// Step 4: Encode to target format
	return c.encode(pcm, to)
}

// decode converts audio bytes to PCM16 samples.
func (c *Converter) decode(audio []byte, format format.AudioFormat) ([]int16, error) {
	switch format.Encoding {
	case "pcm16":
		return codec.BytesToInt16(audio, false), nil

	case "mulaw":
		return codec.MulawDecode(audio), nil

	case "alaw":
		return codec.AlawDecode(audio), nil

	default:
		return nil, ErrUnsupportedEncoding
	}
}

// encode converts PCM16 samples to the target format.
func (c *Converter) encode(pcm []int16, format format.AudioFormat) ([]byte, error) {
	switch format.Encoding {
	case "pcm16":
		return codec.Int16ToBytes(pcm, false), nil

	case "mulaw":
		return codec.MulawEncode(pcm), nil

	case "alaw":
		return codec.AlawEncode(pcm), nil

	default:
		return nil, ErrUnsupportedEncoding
	}
}

// TwilioToOpenAI is a convenience function for Twilio → OpenAI conversion.
// Converts mulaw 8kHz to PCM16 24kHz.
func TwilioToOpenAI(audio []byte) ([]byte, error) {
	return New().Convert(audio, format.Twilio, format.OpenAI)
}

// OpenAIToTwilio is a convenience function for OpenAI → Twilio conversion.
// Converts PCM16 24kHz to mulaw 8kHz.
func OpenAIToTwilio(audio []byte) ([]byte, error) {
	return New().Convert(audio, format.OpenAI, format.Twilio)
}

// TwilioToGemini is a convenience function for Twilio → Gemini conversion.
// Converts mulaw 8kHz to PCM16 16kHz.
func TwilioToGemini(audio []byte) ([]byte, error) {
	return New().Convert(audio, format.Twilio, format.GeminiInput)
}

// GeminiToTwilio is a convenience function for Gemini → Twilio conversion.
// Converts PCM16 24kHz to mulaw 8kHz.
func GeminiToTwilio(audio []byte) ([]byte, error) {
	return New().Convert(audio, format.GeminiOutput, format.Twilio)
}

// StreamConverter wraps Converter for streaming audio conversion.
// It maintains state for processing audio chunks efficiently.
type StreamConverter struct {
	conv       *Converter
	fromFormat format.AudioFormat
	toFormat   format.AudioFormat
}

// NewStreamConverter creates a converter optimized for streaming.
func NewStreamConverter(from, to format.AudioFormat) *StreamConverter {
	return &StreamConverter{
		conv:       New(),
		fromFormat: from,
		toFormat:   to,
	}
}

// Convert converts a chunk of streaming audio.
func (sc *StreamConverter) Convert(audio []byte) ([]byte, error) {
	return sc.conv.Convert(audio, sc.fromFormat, sc.toFormat)
}
