package provider

import "testing"

// mockProvider implements Named for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }

func TestNewClient(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}
	p3 := &mockProvider{name: "provider3"}

	c := NewClient(p1, p2, p3)

	// First provider should be primary
	if c.PrimaryName() != "provider1" {
		t.Errorf("expected primary to be provider1, got %s", c.PrimaryName())
	}

	// Rest should be fallbacks
	fallbacks := c.FallbackNames()
	if len(fallbacks) != 2 {
		t.Errorf("expected 2 fallbacks, got %d", len(fallbacks))
	}
	if fallbacks[0] != "provider2" || fallbacks[1] != "provider3" {
		t.Errorf("unexpected fallbacks: %v", fallbacks)
	}
}

func TestNewClientEmpty(t *testing.T) {
	c := NewClient[*mockProvider]()

	if c.PrimaryName() != "" {
		t.Errorf("expected empty primary, got %s", c.PrimaryName())
	}

	p, ok := c.Primary()
	if ok || p != nil {
		t.Error("expected Primary() to return false for empty client")
	}
}

func TestSetPrimary(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}

	c := NewClient(p1, p2)
	c.SetPrimary("provider2")

	if c.PrimaryName() != "provider2" {
		t.Errorf("expected primary to be provider2, got %s", c.PrimaryName())
	}
}

func TestSetFallbacks(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}
	p3 := &mockProvider{name: "provider3"}

	c := NewClient(p1, p2, p3)
	c.SetFallbacks("provider3", "provider2")

	fallbacks := c.FallbackNames()
	if len(fallbacks) != 2 {
		t.Errorf("expected 2 fallbacks, got %d", len(fallbacks))
	}
	if fallbacks[0] != "provider3" || fallbacks[1] != "provider2" {
		t.Errorf("unexpected fallbacks: %v", fallbacks)
	}
}

func TestProvider(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}

	c := NewClient(p1, p2)

	// Get existing provider
	p, ok := c.Provider("provider1")
	if !ok {
		t.Error("expected to find provider1")
	}
	if p.Name() != "provider1" {
		t.Errorf("expected provider1, got %s", p.Name())
	}

	// Get non-existing provider
	_, ok = c.Provider("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent provider")
	}
}

func TestPrimary(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}

	c := NewClient(p1, p2)

	p, ok := c.Primary()
	if !ok {
		t.Error("expected to find primary provider")
	}
	if p.Name() != "provider1" {
		t.Errorf("expected provider1, got %s", p.Name())
	}
}

func TestFallbacks(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}
	p3 := &mockProvider{name: "provider3"}

	c := NewClient(p1, p2, p3)

	fallbacks := c.Fallbacks()
	if len(fallbacks) != 2 {
		t.Errorf("expected 2 fallbacks, got %d", len(fallbacks))
	}
	if fallbacks[0].Name() != "provider2" || fallbacks[1].Name() != "provider3" {
		t.Error("unexpected fallback order")
	}
}

func TestAll(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}

	c := NewClient(p1, p2)

	all := c.All()
	if len(all) != 2 {
		t.Errorf("expected 2 providers, got %d", len(all))
	}
	if all["provider1"].Name() != "provider1" {
		t.Error("missing provider1")
	}
	if all["provider2"].Name() != "provider2" {
		t.Error("missing provider2")
	}
}

func TestFallbackNamesImmutability(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}

	c := NewClient(p1, p2)

	fallbacks := c.FallbackNames()
	fallbacks[0] = "modified"

	// Original should be unchanged
	original := c.FallbackNames()
	if original[0] != "provider2" {
		t.Error("FallbackNames should return a copy")
	}
}
