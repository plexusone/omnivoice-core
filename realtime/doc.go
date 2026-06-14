// Package realtime provides a unified interface for real-time voice-to-voice providers.
//
// Real-time providers enable native voice-to-voice conversations with ~100-300ms latency
// by handling audio input and output directly, without separate STT/TTS steps.
//
// # Supported Providers
//
// The following providers implement the [Provider] interface:
//
//   - OpenAI Realtime API (github.com/plexusone/omni-openai/omnivoice/realtime)
//   - Gemini Live API (github.com/plexusone/omni-google/omnivoice)
//
// # Audio Format
//
// Input audio should be PCM16 (signed 16-bit little-endian) at the provider's
// expected sample rate:
//
//   - OpenAI Realtime: 24kHz mono
//   - Gemini Live: 16kHz mono (input), 24kHz mono (output)
//
// Output audio is PCM16 24kHz mono for both providers.
//
// # Usage
//
//	provider := openairealtime.NewProvider(apiKey,
//	    openairealtime.WithVoice("alloy"),
//	    openairealtime.WithInstructions("You are a helpful assistant."),
//	)
//
//	audioIn := make(chan []byte, 100)
//	audioCh, transcriptCh, err := provider.ProcessAudioStream(ctx, audioIn, realtime.ProcessConfig{
//	    OnFunctionCall: func(id, name, args string) (any, error) {
//	        return handleFunction(name, args)
//	    },
//	})
//
//	// Send audio from microphone
//	go func() {
//	    for chunk := range microphoneAudio {
//	        audioIn <- chunk
//	    }
//	    close(audioIn)
//	}()
//
//	// Receive audio and transcripts
//	for {
//	    select {
//	    case audio, ok := <-audioCh:
//	        if !ok {
//	            return
//	        }
//	        playAudio(audio.Audio)
//	    case transcript := <-transcriptCh:
//	        log.Printf("[%s] %s", transcript.Role(), transcript.Text)
//	    }
//	}
//
// # Integration with Telephony
//
// Real-time providers integrate with telephony gateways (Twilio, Telnyx, Plivo)
// by connecting the gateway's audio streams to the provider:
//
//	gateway.OnCall(func(session gateway.Session) {
//	    audioIn := make(chan []byte, 100)
//
//	    // Forward gateway audio to provider
//	    go func() {
//	        for chunk := range session.AudioIn() {
//	            // Convert mulaw 8kHz to PCM16 24kHz
//	            pcm := codec.MulawToPCM16(chunk)
//	            resampled := resample(pcm, 8000, 24000)
//	            audioIn <- resampled
//	        }
//	        close(audioIn)
//	    }()
//
//	    audioCh, _, _ := provider.ProcessAudioStream(ctx, audioIn, config)
//
//	    // Forward provider audio to gateway
//	    for audio := range audioCh {
//	        // Convert PCM16 24kHz to mulaw 8kHz
//	        resampled := resample(audio.Audio, 24000, 8000)
//	        mulaw := codec.PCM16ToMulaw(resampled)
//	        session.SendAudio(mulaw)
//	    }
//	})
package realtime
