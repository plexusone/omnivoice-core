# Raw Call Audio (Audio Mode)

Voice gateways normally run a conversation pipeline on every call: speech
recognition, an LLM, and speech synthesis (text mode), or a voice-to-voice
model (realtime mode). The pipeline listens and answers on its own, which suits
a one-to-one call with an assistant.

Audio mode runs no pipeline. The gateway still answers webhooks, connects the
media stream, and tracks the call, but it hands the call's audio to your
application and plays the audio your application sends. Use it when the
application decides when to listen and when to speak, for example:

- an agent attending a meeting by phone, which records and transcribes
  everyone but speaks only when addressed
- a call recorder or live transcription service
- a custom pipeline that does not fit text or realtime mode

## Enabling Audio Mode

Set the gateway's pipeline mode to `gateway.PipelineModeAudio`. With
[omni-twilio](https://github.com/plexusone/omni-twilio):

```go
import (
    coregateway "github.com/plexusone/omnivoice-core/gateway"
    twiliogateway "github.com/plexusone/omni-twilio/omnivoice/gateway"
)

gw, err := twiliogateway.New(twiliogateway.Config{
    AccountSID:  os.Getenv("TWILIO_ACCOUNT_SID"),
    AuthToken:   os.Getenv("TWILIO_AUTH_TOKEN"),
    PhoneNumber: "+14155550199",
    PublicURL:   "https://your-server.com",
    Mode:        coregateway.PipelineModeAudio,
})
```

No STT, LLM, or TTS configuration is needed in audio mode.

## Using the Session

Sessions of an audio-mode gateway implement `gateway.AudioSession`:

| Method | Purpose |
|--------|---------|
| `AudioFormat()` | Format of the audio in both directions, for example mu-law 8 kHz mono |
| `ReceiveAudio()` | Channel of the remote party's audio; closed when the call ends |
| `SendAudio(audio)` | Play audio into the call |
| `ClearAudio()` | Discard audio sent but not played yet, for example when interrupted |

```go
sess, err := coregateway.MakeCallWithOptions(ctx, gw, "+16695550100",
    callsystem.WithSendDigits("WWWW85012345678#"),
)
if err != nil {
    return err
}
audio, ok := sess.(coregateway.AudioSession)
if !ok {
    return errors.New("gateway is not in audio mode")
}

go func() {
    for chunk := range audio.ReceiveAudio() {
        recorder.Write(chunk) // or send to a streaming STT provider
    }
}()
```

Audio flows once the media stream connects, which the session reports with an
`EventSessionStarted` event. Before then, `SendAudio` returns
`ErrAudioNotConnected`; after the call ends it returns `ErrSessionClosed`. When
the media stream stops, the session sends `EventSessionEnded`, closes the
`ReceiveAudio` channel, and closes itself.

The session type assertion succeeds only in audio mode, so code can check
which mode a gateway runs in by asserting `AudioSession`.

## Audio Format

Audio stays in the provider's telephony format. Convert it with the
`audio/converter` package when a speech provider or recorder needs PCM:

| Provider | Format |
|----------|--------|
| omni-twilio | mu-law, 8 kHz, mono (`format.Twilio`) |
| omni-telnyx | mu-law (PCMU), 8 kHz, mono (`format.Telnyx`) |

On a conference line, `ReceiveAudio` carries one mixed stream of everyone on
the call, not a stream per speaker.

## Provider Support

Audio mode requires the omni-twilio and omni-telnyx releases built on
omnivoice-core v0.18.0. Other gateways do not support it yet.
