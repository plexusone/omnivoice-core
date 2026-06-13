// Package bargein provides barge-in detection for voice conversations.
//
// Barge-in allows users to interrupt the AI agent while it is speaking.
// The detector monitors STT (Speech-to-Text) events for user speech and
// automatically stops TTS (Text-to-Speech) playback when appropriate.
//
// # Interruption Modes
//
// The detector supports multiple interruption modes:
//
//   - ModeImmediate: Stop agent speech immediately when user starts speaking
//   - ModeAfterSentence: Wait for agent to complete current sentence
//   - ModeDisabled: Never interrupt (user must wait for agent to finish)
//
// # Usage
//
//	cfg := bargein.Config{
//	    Mode:                bargein.ModeImmediate,
//	    MinSpeechDurationMs: 200,  // Avoid false triggers from noise
//	    SilenceThresholdMs:  500,
//	}
//
//	detector := bargein.NewDetector(cfg)
//	detector.AttachTTS(ttsPipeline)
//	detector.AttachSTTEvents(sttEvents)
//
//	detector.OnInterrupt(func(event gateway.Event) {
//	    // Handle interruption (e.g., log, cancel pending responses)
//	})
//
//	detector.Start(ctx)
//
// # Integration
//
// The detector integrates with existing omnivoice-core components:
//   - pipeline.TTSPipeline: Stopped when barge-in is detected
//   - stt.StreamEvent: Monitors for EventSpeechStart and EventSpeechEnd
//   - gateway.Event: Emits EventInterruption on barge-in
package bargein
