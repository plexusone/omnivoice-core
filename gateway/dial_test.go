package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/plexusone/omnivoice-core/callsystem"
)

// fakeGateway implements Gateway and records outbound calls.
type fakeGateway struct {
	calledTo string
	opts     []callsystem.CallOption
}

func (g *fakeGateway) Name() ProviderName                { return "fake" }
func (g *fakeGateway) Start(context.Context) error       { return nil }
func (g *fakeGateway) Stop() error                       { return nil }
func (g *fakeGateway) OnCall(CallHandler)                {}
func (g *fakeGateway) GetSession(string) (Session, bool) { return nil, false }
func (g *fakeGateway) ListSessions() []Session           { return nil }
func (g *fakeGateway) MakeCall(_ context.Context, to string) (Session, error) {
	g.calledTo = to
	return nil, nil
}

// fakeOptionGateway also implements OptionCaller.
type fakeOptionGateway struct {
	fakeGateway
}

func (g *fakeOptionGateway) MakeCallWithOptions(_ context.Context, to string, opts ...callsystem.CallOption) (Session, error) {
	g.calledTo = to
	g.opts = opts
	return nil, nil
}

func TestMakeCallWithOptionsNoOptionsUsesMakeCall(t *testing.T) {
	gw := &fakeGateway{}
	if _, err := MakeCallWithOptions(context.Background(), gw, "+15551234567"); err != nil {
		t.Fatalf("MakeCallWithOptions() error: %v", err)
	}
	if gw.calledTo != "+15551234567" {
		t.Fatalf("MakeCall to = %q, want %q", gw.calledTo, "+15551234567")
	}
}

func TestMakeCallWithOptionsUnsupported(t *testing.T) {
	gw := &fakeGateway{}
	_, err := MakeCallWithOptions(context.Background(), gw, "+15551234567", callsystem.WithSendDigits("123#"))
	if !errors.Is(err, ErrCallOptionsUnsupported) {
		t.Fatalf("error = %v, want ErrCallOptionsUnsupported", err)
	}
	if gw.calledTo != "" {
		t.Fatalf("call was placed despite unsupported options")
	}
}

func TestMakeCallWithOptionsPassesOptions(t *testing.T) {
	gw := &fakeOptionGateway{}
	_, err := MakeCallWithOptions(context.Background(), gw, "+15551234567", callsystem.WithSendDigits("123#"))
	if err != nil {
		t.Fatalf("MakeCallWithOptions() error: %v", err)
	}

	got := &callsystem.CallOptions{}
	for _, opt := range gw.opts {
		opt(got)
	}
	if got.SendDigits != "123#" {
		t.Fatalf("SendDigits = %q, want %q", got.SendDigits, "123#")
	}
}
