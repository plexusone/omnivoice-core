// Example: Voice agent in Zoom meetings
//
// This example demonstrates how to add a voice agent to Zoom meetings.
// Due to Zoom SDK complexity, we recommend using Recall.ai as the bot infrastructure.
//
// Architecture with Recall.ai:
//
//	┌──────────────────────────────────────────────────────────────────┐
//	│                        Zoom Meeting                               │
//	│   ┌─────────┐  ┌─────────┐  ┌─────────────────────────────────┐  │
//	│   │  User 1 │  │  User 2 │  │     Recall.ai Bot               │  │
//	│   └─────────┘  └─────────┘  └────────────────┬────────────────┘  │
//	└──────────────────────────────────────────────┼────────────────────┘
//	                                               │
//	                                   WebSocket   │ (Audio stream)
//	                                               ▼
//	                              ┌─────────────────────────────────┐
//	                              │        OmniVoice Agent          │
//	                              │  ┌─────┐ ┌─────┐ ┌─────┐        │
//	                              │  │ STT │→│ LLM │→│ TTS │        │
//	                              │  └─────┘ └─────┘ └─────┘        │
//	                              └─────────────────────────────────┘
//
// Why Recall.ai instead of native Zoom SDK:
// - Zoom doesn't provide a simple API for real-time audio
// - Native SDK requires one process per meeting, complex scaling
// - Recall.ai handles Zoom SDK complexity, provides WebSocket stream
// - You can be live in hours instead of months
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Example: Join a Zoom meeting using Recall.ai
	meetingURL := os.Getenv("ZOOM_MEETING_URL")
	if meetingURL == "" {
		log.Fatal("ZOOM_MEETING_URL environment variable required")
	}

	// TODO: Implement with OmniVoice + Recall.ai
	//
	// Option 1: Use Recall.ai REST API
	//
	// 1. Create bot via Recall.ai API:
	//    POST https://api.recall.ai/api/v1/bot
	//    {
	//      "meeting_url": "https://zoom.us/j/123...",
	//      "bot_name": "AI Assistant",
	//      "real_time_transcription": { "enabled": true },
	//      "real_time_media": { "websocket_audio_output": true }
	//    }
	//
	// 2. Connect to Recall.ai WebSocket for audio stream
	// 3. Create OmniVoice agent session
	// 4. Wire audio in/out through the session
	//
	// Option 2: Use Recall.ai Go SDK (when available)
	//
	// recall := recallai.NewClient(apiKey)
	// bot, _ := recall.CreateBot(ctx, recallai.BotConfig{
	//     MeetingURL: meetingURL,
	//     BotName:    "AI Assistant",
	// })
	//
	// session, _ := agentProvider.CreateSession(ctx, agent.Config{...})
	// bot.AudioStream().Pipe(session)
	//
	// Option 3: Self-hosted Zoom SDK (complex, not recommended)
	//
	// Requires:
	// - Running Zoom SDK bot on Linux VM
	// - Managing one instance per meeting
	// - Handling SDK raw audio callbacks
	// - Months of engineering work

	safeMeetingURL := strings.ReplaceAll(strings.ReplaceAll(meetingURL, "\n", ""), "\r", "")
	log.Printf("Would join meeting: %s", safeMeetingURL) //nolint:gosec // G706: sanitized
	log.Println("Implementation requires Recall.ai API key")
	log.Println("See: https://www.recall.ai/")

	<-ctx.Done()
}
