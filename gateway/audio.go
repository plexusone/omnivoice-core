package gateway

import (
	"errors"

	"github.com/plexusone/omnivoice-core/audio/format"
)

var (
	// ErrSessionClosed is returned by AudioSession methods after the call
	// ends.
	ErrSessionClosed = errors.New("gateway session closed")

	// ErrAudioNotConnected is returned by AudioSession methods that send
	// audio before the call's media stream connects.
	ErrAudioNotConnected = errors.New("call audio is not connected yet")
)

// AudioSession is implemented by sessions of gateways running in
// PipelineModeAudio. It carries the call's audio to and from the application
// in the provider's telephony format, with no speech recognition or synthesis
// in between.
//
//	sess, err := gateway.MakeCallWithOptions(ctx, gw, to)
//	if err != nil {
//		return err
//	}
//	audio, ok := sess.(gateway.AudioSession)
//	if !ok {
//		return errors.New("gateway is not in audio mode")
//	}
//	for chunk := range audio.ReceiveAudio() {
//		// transcribe, record, or analyze chunk
//	}
type AudioSession interface {
	Session

	// AudioFormat returns the format of the audio in both directions, for
	// example format.Twilio (mu-law, 8 kHz, mono).
	AudioFormat() format.AudioFormat

	// ReceiveAudio returns the remote party's audio as it arrives. The
	// channel is closed when the call ends. Audio arrives only after the
	// media stream connects (EventSessionStarted).
	ReceiveAudio() <-chan []byte

	// SendAudio queues audio to play into the call. It returns
	// ErrAudioNotConnected before the media stream connects and
	// ErrSessionClosed after the call ends.
	SendAudio(audio []byte) error

	// ClearAudio discards audio queued by SendAudio that has not played yet,
	// for example when the agent is interrupted.
	ClearAudio() error
}
