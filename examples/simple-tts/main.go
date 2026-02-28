// Example: Simple TTS with provider fallback
//
// This example demonstrates basic text-to-speech usage with
// automatic fallback between providers.
package main

import (
	"context"
	"fmt"

	"github.com/plexusone/omnivoice-core/tts"
)

func main() {
	ctx := context.Background()

	// TODO: Initialize providers (implementation not yet available)
	//
	// elevenLabs := elevenlabs.New(os.Getenv("ELEVENLABS_API_KEY"))
	// awsPolly := polly.New(session)
	// googleTTS := google.New(ctx, option.WithCredentialsFile("creds.json"))

	// Create client with multiple providers
	// Primary: ElevenLabs (best quality)
	// Fallback 1: AWS Polly (reliable)
	// Fallback 2: Google TTS (wide language support)
	//
	// client := tts.NewClient(elevenLabs, awsPolly, googleTTS)

	// Example synthesis
	text := "Hello! This is a demonstration of OmniVoice text-to-speech capabilities."

	config := tts.SynthesisConfig{
		VoiceID:      "rachel", // ElevenLabs voice
		OutputFormat: "mp3",
		SampleRate:   44100,
	}

	// Synthesize with automatic fallback
	// If ElevenLabs fails, tries AWS Polly, then Google TTS
	//
	// result, err := client.Synthesize(ctx, text, config)
	// if err != nil {
	//     log.Fatalf("All providers failed: %v", err)
	// }
	//
	// // Save to file
	// os.WriteFile("output.mp3", result.Audio, 0644)
	// fmt.Printf("Generated %d bytes, duration: %dms\n",
	//     len(result.Audio), result.DurationMs)

	// Streaming example (for real-time playback)
	//
	// stream, err := client.SynthesizeStream(ctx, text, config)
	// if err != nil {
	//     log.Fatalf("Failed to start stream: %v", err)
	// }
	//
	// for chunk := range stream {
	//     if chunk.Error != nil {
	//         log.Printf("Stream error: %v", chunk.Error)
	//         break
	//     }
	//     // Play audio chunk in real-time
	//     playAudio(chunk.Audio)
	// }

	fmt.Println("TTS Example")
	fmt.Printf("Text: %s\n", text)
	fmt.Printf("Config: voice=%s, format=%s, rate=%d\n",
		config.VoiceID, config.OutputFormat, config.SampleRate)
	fmt.Println("\nProvider implementations coming soon!")

	// Placeholder to avoid unused import errors
	_ = ctx
}
