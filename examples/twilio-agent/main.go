// Example: Voice agent handling inbound Twilio calls
//
// This example demonstrates how to wire an OmniVoice agent to handle
// incoming phone calls via Twilio ConversationRelay.
//
// Architecture:
//
//	┌──────────┐        ┌─────────────────┐        ┌───────────────────┐
//	│  Caller  │◄──────►│     Twilio      │◄──────►│   OmniVoice       │
//	│  (PSTN)  │  PSTN  │ ConversationRelay│  WS   │   Voice Agent     │
//	└──────────┘        └─────────────────┘        └───────────────────┘
//
// Flow:
// 1. Caller dials your Twilio phone number
// 2. Twilio webhook hits your server
// 3. Server returns TwiML connecting to ConversationRelay
// 4. ConversationRelay opens WebSocket to your agent
// 5. Agent receives audio, processes with STT → LLM → TTS
// 6. Agent sends audio back through the same WebSocket
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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

	// Start HTTP server for Twilio webhooks
	http.HandleFunc("/voice/inbound", handleInboundCall)
	http.HandleFunc("/voice/status", handleCallStatus)

	// Start WebSocket server for ConversationRelay
	http.HandleFunc("/ws/agent", handleAgentWebSocket)

	addr := ":8080"
	log.Printf("Starting server on %s", addr)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	server.Close()
}

// handleInboundCall handles incoming Twilio webhook for new calls.
// Returns TwiML to connect the call to ConversationRelay.
func handleInboundCall(w http.ResponseWriter, r *http.Request) {
	// Get caller info from Twilio webhook
	from := r.FormValue("From")
	to := r.FormValue("To")
	callSID := r.FormValue("CallSid")

	// Sanitize values before logging to prevent log injection via newlines
	safeFrom := strings.ReplaceAll(strings.ReplaceAll(from, "\n", ""), "\r", "")
	safeTo := strings.ReplaceAll(strings.ReplaceAll(to, "\n", ""), "\r", "")
	safeCallSID := strings.ReplaceAll(strings.ReplaceAll(callSID, "\n", ""), "\r", "")

	log.Printf("Incoming call: %s -> %s (SID: %s)", safeFrom, safeTo, safeCallSID)

	// Return TwiML connecting to ConversationRelay
	// This tells Twilio to open a WebSocket to our agent
	wsURL := fmt.Sprintf("wss://%s/ws/agent?callSid=%s", r.Host, callSID)

	twiml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Connect>
        <ConversationRelay
            url="%s"
            voice="Google.en-US-Neural2-F"
            transcriptionProvider="deepgram"
            speechModel="nova-2"
            interruptible="true"
            interruptByDtmf="true"
            dtmfDetection="true">
            <Parameter name="caller" value="%s"/>
        </ConversationRelay>
    </Connect>
</Response>`, wsURL, from)

	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write([]byte(twiml)); err != nil {
		slog.Error("failed to write TwiML response", "error", err, "callSid", callSID)
	}
}

// handleCallStatus handles Twilio status callbacks.
func handleCallStatus(w http.ResponseWriter, r *http.Request) {
	callSID := r.FormValue("CallSid")
	status := r.FormValue("CallStatus")

	// Sanitize values before logging to prevent log injection via newlines
	safeCallSID := strings.ReplaceAll(strings.ReplaceAll(callSID, "\n", ""), "\r", "")
	safeStatus := strings.ReplaceAll(strings.ReplaceAll(status, "\n", ""), "\r", "")

	log.Printf("Call status update: %s -> %s", safeCallSID, safeStatus)
	w.WriteHeader(http.StatusOK)
}

// handleAgentWebSocket handles the ConversationRelay WebSocket connection.
// This is where the voice agent receives and sends audio.
func handleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	callSID := r.URL.Query().Get("callSid")
	safeCallSID := strings.ReplaceAll(strings.ReplaceAll(callSID, "\n", ""), "\r", "")
	log.Printf("Agent WebSocket connected for call: %s", safeCallSID)

	// TODO: Implement WebSocket handling with OmniVoice
	//
	// 1. Upgrade HTTP connection to WebSocket
	// 2. Create agent session with STT + LLM + TTS
	// 3. Receive audio frames from Twilio
	// 4. Send audio to STT for transcription
	// 5. Send transcript to LLM for response
	// 6. Send response to TTS for synthesis
	// 7. Send audio frames back to Twilio
	//
	// Example with OmniVoice (pseudo-code):
	//
	// session, _ := agentProvider.CreateSession(ctx, agent.Config{
	//     SystemPrompt: "You are a helpful assistant...",
	//     VoiceID:      "elevenlabs-rachel",
	//     STTProvider:  "deepgram",
	//     TTSProvider:  "elevenlabs",
	//     LLMProvider:  "anthropic",
	//     LLMModel:     "claude-sonnet-4-20250514",
	// })
	//
	// wsConn := websocket.Accept(w, r, nil)
	// transport := websocket.NewTransport(wsConn)
	// transport.Connect(ctx, session)
	//
	// <-session.Events() // Wait for session end

	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
