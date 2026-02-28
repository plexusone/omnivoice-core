// Package providertest provides conformance tests for CallSystem provider implementations.
//
// Provider implementations can use this package to verify they correctly implement
// the callsystem.CallSystem interface with consistent behavior.
//
// Basic usage:
//
//	func TestConformance(t *testing.T) {
//	    p, err := New(WithAccountSID(sid), WithAuthToken(token), WithPhoneNumber(phone))
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    providertest.RunAll(t, providertest.Config{
//	        Provider:        p,
//	        SkipIntegration: phone == "",
//	        TestPhoneNumber: phone,
//	        TestFromNumber:  phone,
//	    })
//	}
package providertest

import (
	"context"
	"testing"
	"time"

	"github.com/plexusone/omnivoice/callsystem"
)

// Config configures the callsystem conformance test suite.
type Config struct {
	// Provider is the CallSystem provider implementation to test.
	Provider callsystem.CallSystem

	// SkipIntegration skips tests that require real API calls.
	SkipIntegration bool

	// TestPhoneNumber is the number to call for integration tests (E.164 format).
	// Required if SkipIntegration is false.
	TestPhoneNumber string

	// TestFromNumber is the caller ID for outbound calls (E.164 format).
	// Required if SkipIntegration is false.
	TestFromNumber string

	// Timeout for individual test operations.
	// Defaults to 30 seconds if zero.
	Timeout time.Duration
}

// withDefaults returns a copy of Config with default values applied.
func (c Config) withDefaults() Config {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	return c
}

// RunAll runs all conformance tests for a CallSystem provider.
func RunAll(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Interface", func(t *testing.T) {
		RunInterfaceTests(t, cfg)
	})

	t.Run("Behavior", func(t *testing.T) {
		RunBehaviorTests(t, cfg)
	})

	if !cfg.SkipIntegration {
		t.Run("Integration", func(t *testing.T) {
			RunIntegrationTests(t, cfg)
		})
	}
}

// RunInterfaceTests runs only interface compliance tests.
// These tests verify the provider correctly implements the interface contract
// and do not require API credentials.
func RunInterfaceTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Name", func(t *testing.T) { testName(t, cfg) })
	t.Run("Configure", func(t *testing.T) { testConfigure(t, cfg) })
	t.Run("OnIncomingCall", func(t *testing.T) { testOnIncomingCall(t, cfg) })
	t.Run("ListCalls_Empty", func(t *testing.T) { testListCallsEmpty(t, cfg) })
}

// RunBehaviorTests runs only behavioral contract tests.
func RunBehaviorTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Close_Idempotent", func(t *testing.T) { testCloseIdempotent(t, cfg) })
	t.Run("MakeCall_EmptyTo", func(t *testing.T) { testMakeCallEmptyTo(t, cfg) })
}

// RunIntegrationTests runs only integration tests (requires API credentials).
func RunIntegrationTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	if cfg.SkipIntegration {
		t.Skip("integration tests skipped")
	}
	t.Run("MakeCall_Lifecycle", func(t *testing.T) { testMakeCallLifecycle(t, cfg) })
}

// Interface Tests

func testName(t *testing.T, cfg Config) {
	t.Helper()
	name := cfg.Provider.Name()
	if name == "" {
		t.Error("Name() returned empty string")
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			t.Errorf("Name() contains invalid character %q; should be lowercase alphanumeric with hyphens/underscores", r)
		}
	}
}

func testConfigure(t *testing.T, cfg Config) {
	t.Helper()
	err := cfg.Provider.Configure(callsystem.CallSystemConfig{})
	if err != nil {
		t.Errorf("Configure() error: %v", err)
	}
}

func testOnIncomingCall(t *testing.T, cfg Config) {
	t.Helper()
	// Should not panic when setting a handler
	cfg.Provider.OnIncomingCall(func(call callsystem.Call) error {
		return nil
	})
}

func testListCallsEmpty(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	calls, err := cfg.Provider.ListCalls(ctx)
	if err != nil {
		t.Fatalf("ListCalls() error: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("ListCalls() on fresh provider returned %d calls, want 0", len(calls))
	}
}

// Behavior Tests

func testCloseIdempotent(t *testing.T, cfg Config) {
	t.Helper()
	// Multiple Close calls should not panic
	_ = cfg.Provider.Close()
	err := cfg.Provider.Close()
	if err != nil {
		t.Logf("Second Close() returned error (acceptable): %v", err)
	}
}

func testMakeCallEmptyTo(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.SkipIntegration {
		t.Skip("skipping behavior test that requires API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Empty destination should return an error, not panic
	_, err := cfg.Provider.MakeCall(ctx, "")
	if err == nil {
		t.Error("MakeCall(\"\") should return error for empty destination")
	}
}

// Integration Tests

func testMakeCallLifecycle(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if cfg.TestPhoneNumber == "" {
		t.Skip("TestPhoneNumber not set")
	}
	if cfg.TestFromNumber == "" {
		t.Skip("TestFromNumber not set")
	}

	// Make an outbound call
	call, err := cfg.Provider.MakeCall(ctx, cfg.TestPhoneNumber,
		callsystem.WithFrom(cfg.TestFromNumber),
	)
	if err != nil {
		t.Fatalf("MakeCall() error: %v", err)
	}

	// Verify call fields
	if call.ID() == "" {
		t.Error("Call.ID() is empty")
	}
	if call.Direction() != callsystem.Outbound {
		t.Errorf("Call.Direction() = %q, want %q", call.Direction(), callsystem.Outbound)
	}
	if call.From() == "" {
		t.Error("Call.From() is empty")
	}
	if call.To() == "" {
		t.Error("Call.To() is empty")
	}

	t.Logf("MakeCall() created call %s (status: %s)", call.ID(), call.Status())

	// Give the call time to be processed
	time.Sleep(2 * time.Second)

	// Verify call appears in list
	calls, err := cfg.Provider.ListCalls(ctx)
	if err != nil {
		t.Fatalf("ListCalls() error: %v", err)
	}
	found := false
	for _, c := range calls {
		if c.ID() == call.ID() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListCalls() does not contain call %s", call.ID())
	}

	// Hangup the call
	err = call.Hangup(ctx)
	if err != nil {
		t.Fatalf("Hangup() error: %v", err)
	}

	t.Logf("Call %s hung up (status: %s)", call.ID(), call.Status())

	// Verify status is ended
	status := call.Status()
	if status != callsystem.StatusEnded && status != callsystem.StatusFailed {
		t.Logf("Call status after hangup: %s (may not be immediately updated)", status)
	}
}
