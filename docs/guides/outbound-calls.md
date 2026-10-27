# Outbound Calls and Keypad Digits

This guide covers placing outbound calls with a call system or voice gateway,
the options a call accepts, and sending keypad tones (DTMF) once the call is
answered, for example to join a meeting's phone dial-in line.

## Placing a Call

A `callsystem.CallSystem` places calls with `MakeCall`, which takes the
destination in E.164 form and any number of call options:

```go
import "github.com/plexusone/omnivoice-core/callsystem"

call, err := calls.MakeCall(ctx, "+14155550100",
    callsystem.WithFrom("+14155550199"),
    callsystem.WithTimeout(30*time.Second),
)
if err != nil {
    return err
}
```

| Option | Effect |
|--------|--------|
| `WithFrom(number)` | Caller ID; defaults to the provider's configured number |
| `WithTimeout(d)` | How long the call rings before giving up |
| `WithMachineDetection()` | Detect answering machines |
| `WithRecording()` | Record the call |
| `WithWhisper(message)` | Message for the agent before the call connects |
| `WithAgent(config)` | Attach a voice agent to the call |
| `WithStatusCallback(url)` | Webhook URL for call status updates |
| `WithObserver(observer)` | Receive voice events for observability |
| `WithSendDigits(digits)` | Keypad tones to send once the call is answered |

Providers apply the options they support.

## Sending Keypad Digits

`WithSendDigits` sends keypad tones after the call is answered. Use it to
navigate phone menus, enter extensions, or join a conference line with a
meeting ID and passcode.

Digits use dial-string notation:

| Character | Meaning |
|-----------|---------|
| `0`-`9`, `*`, `#` | Keypad tones |
| `A`-`D` | Extended DTMF tones |
| `w` | Wait half a second |
| `W` | Wait one second |

Pauses let a phone menu finish its prompt before the next entry. For example,
joining a conference line that asks for a meeting ID and then a passcode:

```go
// Wait 4 seconds, enter the meeting ID and #, wait 8 seconds, then enter
// * and the passcode and #.
digits := "WWWW85012345678#WWWWWWWW*482913#"

call, err := calls.MakeCall(ctx, "+16695550100", callsystem.WithSendDigits(digits))
```

`MakeCall` returns an error for any other character. Validate digits from user
input before calling with `ValidateSendDigits`:

```go
if err := callsystem.ValidateSendDigits(digits); err != nil {
    return fmt.Errorf("invalid keypad digits: %w", err)
}
```

For meeting dial-in, [OmniMeet](https://github.com/plexusone/omnimeet-core)'s
`invite` and `dialin` packages read the dial-in number, meeting ID, and
passcode from a Zoom, Google Meet, or Teams invitation and build this string
for you.

## Calls Through a Voice Gateway

A `gateway.Gateway` handles the media stream and voice pipeline for a call.
Its `MakeCall` method takes no options, so that existing gateways keep
working. Gateways that accept options also implement `gateway.OptionCaller`.

Use `gateway.MakeCallWithOptions` to place a call with options through any
gateway:

```go
import (
    "github.com/plexusone/omnivoice-core/callsystem"
    "github.com/plexusone/omnivoice-core/gateway"
)

session, err := gateway.MakeCallWithOptions(ctx, gw, "+16695550100",
    callsystem.WithSendDigits(digits),
)
if errors.Is(err, gateway.ErrCallOptionsUnsupported) {
    // This gateway cannot send keypad digits.
}
```

Without options it calls `gw.MakeCall`. With options it returns
`ErrCallOptionsUnsupported` when the gateway does not implement
`OptionCaller`, rather than placing the call and silently dropping the digits.

## Provider Support

| Provider | Keypad digits | How |
|----------|---------------|-----|
| [omni-twilio](https://github.com/plexusone/omni-twilio) | Yes | Played with `<Play digits>` before the media stream connects |
| [omni-telnyx](https://github.com/plexusone/omni-telnyx) | Yes | Sent with the `send_dtmf` command when the call is answered |

Support in these providers requires their releases built on omnivoice-core
v0.17.0. Other providers return `ErrCallOptionsUnsupported` from
`gateway.MakeCallWithOptions`.

## Adding Support to a Provider

To support keypad digits in a call system provider:

1. Read `CallOptions.SendDigits` in `MakeCall` and validate it with
   `callsystem.ValidateSendDigits` before placing the call.
2. Send the digits once the call is answered, using the provider's own
   mechanism. Convert the notation if the provider differs; for example,
   Twilio's `<Play digits>` supports only `w`, so `W` becomes `ww`.
3. Make sure the digits are sent once, even if the answer webhook is retried.

In the provider's gateway, implement `OptionCaller` and pass the options
through to the call system:

```go
var _ gateway.OptionCaller = (*Gateway)(nil)

func (g *Gateway) MakeCall(ctx context.Context, to string) (gateway.Session, error) {
    return g.MakeCallWithOptions(ctx, to)
}

func (g *Gateway) MakeCallWithOptions(ctx context.Context, to string, opts ...callsystem.CallOption) (gateway.Session, error) {
    call, err := g.callSystem.MakeCall(ctx, to, opts...)
    if err != nil {
        return nil, err
    }
    return g.createSession(call), nil
}
```
