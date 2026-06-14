// Package converter provides audio format conversion for voice gateways.
//
// It bridges telephony audio formats (mulaw 8kHz) with realtime LLM providers
// (PCM16 16kHz/24kHz) by handling both codec conversion and sample rate conversion.
//
// # Supported Formats
//
//   - Telephony: mulaw 8kHz mono (Twilio, Telnyx, Vonage, Plivo)
//   - OpenAI Realtime: PCM16 24kHz mono
//   - Gemini Live: PCM16 16kHz input, 24kHz output
//
// # Usage
//
//	conv := converter.New()
//
//	// Twilio → OpenAI (mulaw 8kHz → PCM16 24kHz)
//	pcm24k, err := conv.Convert(twilioAudio, gateway.AudioFormatTwilio, gateway.AudioFormatOpenAI)
//
//	// OpenAI → Twilio (PCM16 24kHz → mulaw 8kHz)
//	mulaw8k, err := conv.Convert(openaiAudio, gateway.AudioFormatOpenAI, gateway.AudioFormatTwilio)
package converter
