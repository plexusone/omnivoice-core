// Package providertest provides conformance tests for Transport provider implementations.
//
// Provider implementations can use this package to verify they correctly implement
// the transport.Transport and transport.TelephonyTransport interfaces with consistent behavior.
//
// Basic usage:
//
//	func TestConformance(t *testing.T) {
//	    p, err := New(WithAccountSID(sid), WithAuthToken(token))
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    providertest.RunAll(t, providertest.Config{
//	        Provider:        p,
//	        SkipIntegration: sid == "",
//	    })
//	}
package providertest

import (
	"context"
	"testing"
	"time"

	"github.com/agentplexus/omnivoice/transport"
)

// Config configures the transport conformance test suite.
type Config struct {
	// Provider is the transport provider implementation to test.
	Provider transport.Transport

	// TelephonyProvider is optional; set if provider implements TelephonyTransport.
	// If nil, telephony-specific tests are skipped.
	TelephonyProvider transport.TelephonyTransport

	// SkipIntegration skips tests that require a live transport connection.
	SkipIntegration bool

	// TestServerAddr is the address for integration tests that need a running server.
	TestServerAddr string

	// Timeout for individual test operations.
	// Defaults to 10 seconds if zero.
	Timeout time.Duration
}

// withDefaults returns a copy of Config with default values applied.
func (c Config) withDefaults() Config {
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	return c
}

// RunAll runs all conformance tests for a transport provider.
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
// and do not require live connections.
func RunInterfaceTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Name", func(t *testing.T) { testName(t, cfg) })
	t.Run("Protocol", func(t *testing.T) { testProtocol(t, cfg) })
	t.Run("Listen", func(t *testing.T) { testListen(t, cfg) })
	t.Run("Connect_Unsupported", func(t *testing.T) { testConnectUnsupported(t, cfg) })
	t.Run("Close", func(t *testing.T) { testClose(t, cfg) })
}

// RunBehaviorTests runs only behavioral contract tests.
func RunBehaviorTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	t.Run("Close_Idempotent", func(t *testing.T) { testCloseIdempotent(t, cfg) })
	t.Run("Listen_MultiplePaths", func(t *testing.T) { testListenMultiplePaths(t, cfg) })
}

// RunIntegrationTests runs only integration tests (requires live transport).
func RunIntegrationTests(t *testing.T, cfg Config) {
	t.Helper()
	cfg = cfg.withDefaults()

	if cfg.SkipIntegration {
		t.Skip("integration tests skipped")
	}
	// Integration tests for transport require a live WebSocket server or
	// similar infrastructure. These are typically tested through the
	// callsystem integration tests which exercise the full call lifecycle.
	t.Log("Transport integration tests are exercised via callsystem integration tests")
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

func testProtocol(t *testing.T, cfg Config) {
	t.Helper()
	protocol := cfg.Provider.Protocol()
	if protocol == "" {
		t.Error("Protocol() returned empty string")
	}
}

func testListen(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	ch, err := cfg.Provider.Listen(ctx, "/test-listen")
	if err != nil {
		t.Fatalf("Listen() error: %v", err)
	}
	if ch == nil {
		t.Error("Listen() returned nil channel")
	}
}

func testConnectUnsupported(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Many transport providers don't support outbound connections.
	// This test verifies they return a clear error instead of panicking.
	conn, err := cfg.Provider.Connect(ctx, "ws://localhost:0/test", transport.Config{})
	if err == nil && conn == nil {
		t.Error("Connect() returned nil connection without error")
	}
	// Either an error or a valid connection is acceptable
	if conn != nil {
		_ = conn.Close()
	}
}

func testClose(t *testing.T, cfg Config) {
	t.Helper()
	// Close on a fresh provider should not error
	err := cfg.Provider.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
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

func testListenMultiplePaths(t *testing.T, cfg Config) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	ch1, err := cfg.Provider.Listen(ctx, "/path-1")
	if err != nil {
		t.Fatalf("Listen(/path-1) error: %v", err)
	}
	ch2, err := cfg.Provider.Listen(ctx, "/path-2")
	if err != nil {
		t.Fatalf("Listen(/path-2) error: %v", err)
	}

	if ch1 == nil || ch2 == nil {
		t.Error("Listen() returned nil channel for one of the paths")
	}
}
