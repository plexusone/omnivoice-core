# Barge-in Detection

The `bargein` package provides detection and handling of user interruptions during agent speech, enabling natural conversational flow.

## Overview

Barge-in (also called "interrupt" or "cut-in") occurs when a user starts speaking while the AI agent is still talking. Proper barge-in handling is essential for natural conversation:

- **Without barge-in**: User must wait for agent to finish, feels robotic
- **With barge-in**: User can interrupt naturally, feels conversational

## Configuration

### InterruptionMode

The detector supports three interruption modes:

```go
type InterruptionMode string

const (
    // ModeImmediate interrupts as soon as user speech is detected
    ModeImmediate InterruptionMode = "immediate"

    // ModeAfterSentence waits for agent to complete current sentence
    ModeAfterSentence InterruptionMode = "after_sentence"

    // ModeDisabled ignores user speech during agent output
    ModeDisabled InterruptionMode = "disabled"
)
```

### Config

```go
type Config struct {
    // Mode controls when to interrupt agent speech
    Mode InterruptionMode

    // MinSpeechDurationMs is minimum speech duration to trigger barge-in
    // Helps filter out noise and brief sounds (default: 200ms)
    MinSpeechDurationMs int

    // SilenceThresholdMs is silence duration before considering speech ended
    // Used for ModeAfterSentence (default: 500ms)
    SilenceThresholdMs int
}
```

## Usage

### Basic Setup

```go
import "github.com/plexusone/omnivoice-core/bargein"

// Create detector with immediate interruption
detector := bargein.New(bargein.Config{
    Mode:                bargein.ModeImmediate,
    MinSpeechDurationMs: 200,
})

// Attach TTS pipeline for interruption
detector.AttachTTS(ttsPipeline)

// Attach STT events for speech detection
detector.AttachSTTEvents(sttEvents)

// Set interruption handler
detector.OnInterrupt(func(event gateway.Event) {
    log.Printf("User interrupted at %v", event.Timestamp)
})

// Start detection
if err := detector.Start(ctx); err != nil {
    log.Fatal(err)
}
defer detector.Stop()
```

### With Voice Gateway

```go
import (
    "github.com/plexusone/omnivoice-core/bargein"
    "github.com/plexusone/omnivoice-core/gateway"
)

gw := gateway.New(config)
detector := bargein.New(bargein.Config{
    Mode: bargein.ModeImmediate,
})

// Wire up the detector
gw.OnEvent(func(event gateway.Event) {
    switch event.Type {
    case gateway.EventUserSpeechStart:
        detector.HandleSpeechStart()
    case gateway.EventUserSpeechEnd:
        detector.HandleSpeechEnd()
    }
})

detector.OnInterrupt(func(event gateway.Event) {
    // Stop current TTS playback
    gw.StopTTS()

    // Optionally notify LLM of interruption
    agent.OnInterrupt(event)
})
```

## How It Works

### Immediate Mode

```
Agent speaking: "Hello, how can I help you today—"
User starts:    "I need—"
                     ↓
              [INTERRUPT]
                     ↓
Agent stops, user continues: "—to check my balance"
```

1. Detector monitors STT events for `SpeechStart`
2. If agent is speaking and user starts speaking:
   - Wait for `MinSpeechDurationMs` to filter noise
   - Trigger interrupt and stop TTS

### After-Sentence Mode

```
Agent speaking: "I can help with billing questions."
User starts:              "Actually—"
                          (waits for sentence end)
                                    ↓
                              [INTERRUPT]
```

1. Detector monitors for both speech start and end
2. If user speaks during agent turn:
   - Mark pending interrupt
   - Wait for natural pause (`SilenceThresholdMs`)
   - Stop TTS at sentence boundary

## Integration with TTS Pipeline

The detector integrates with the TTS pipeline to know when the agent is speaking:

```go
type TTSPipeline interface {
    IsActive() bool
    Stop()
}
```

When an interrupt is triggered:

1. Detector checks `ttsPipeline.IsActive()`
2. If active, calls `ttsPipeline.Stop()`
3. Fires the `OnInterrupt` callback

## Best Practices

1. **Use ModeImmediate for most cases** - Feels most natural
2. **Set MinSpeechDurationMs to 200-300ms** - Filters coughs/noise
3. **Handle interrupts gracefully** - Don't lose context
4. **Test with real audio** - Synthetic tests miss edge cases

## Troubleshooting

### False Positives (Too Many Interrupts)

- Increase `MinSpeechDurationMs` to 300-500ms
- Check STT VAD sensitivity settings
- Ensure proper echo cancellation

### Missed Interrupts

- Decrease `MinSpeechDurationMs`
- Check STT is detecting speech starts properly
- Verify TTS pipeline connection

### Choppy Audio After Interrupt

- Add brief delay before resuming
- Clear audio buffers properly
- Check for race conditions in TTS stop
