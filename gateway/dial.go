package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/plexusone/omnivoice-core/callsystem"
)

// ErrCallOptionsUnsupported is returned by MakeCallWithOptions when call
// options are given but the gateway does not implement OptionCaller.
var ErrCallOptionsUnsupported = errors.New("gateway does not support call options on outbound calls")

// OptionCaller is an optional interface for gateways whose outbound calls
// accept callsystem call options, such as keypad digits to send once the
// call is answered (callsystem.WithSendDigits).
//
// It is separate from Gateway so that existing gateways keep compiling;
// use MakeCallWithOptions to call it through a Gateway value.
type OptionCaller interface {
	// MakeCallWithOptions initiates an outbound call with call options.
	MakeCallWithOptions(ctx context.Context, to string, opts ...callsystem.CallOption) (Session, error)
}

// MakeCallWithOptions places an outbound call through gw. Without options it
// calls gw.MakeCall. With options it requires gw to implement OptionCaller and
// returns ErrCallOptionsUnsupported otherwise, rather than silently dropping
// options such as keypad digits.
func MakeCallWithOptions(ctx context.Context, gw Gateway, to string, opts ...callsystem.CallOption) (Session, error) {
	if len(opts) == 0 {
		return gw.MakeCall(ctx, to)
	}
	oc, ok := gw.(OptionCaller)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCallOptionsUnsupported, gw.Name())
	}
	return oc.MakeCallWithOptions(ctx, to, opts...)
}
