package engine

import (
	"context"
	"testing"
)

type stubAdapter struct {
	src       Source
	available bool
	caps      Capabilities
}

func (s *stubAdapter) Source() Source                   { return s.src }
func (s *stubAdapter) Available() bool                  { return s.available }
func (s *stubAdapter) Capabilities() Capabilities       { return s.caps }
func (s *stubAdapter) Analyze(_ context.Context, _ Request) (Response, error) {
	return Response{Source: s.src}, nil
}

func TestRegistryReturnsPrimaryWhenAvailable(t *testing.T) {
	pikafish := &stubAdapter{src: SourcePikafish, available: true}
	fairy := &stubAdapter{src: SourceFairyStockfish, available: true}
	registry := NewRegistry(pikafish, fairy)

	primary, ok := registry.Primary()
	if !ok {
		t.Fatalf("expected primary, got none")
	}
	if primary.Source() != SourcePikafish {
		t.Errorf("primary source = %q, want pikafish", primary.Source())
	}
}

func TestRegistryFallsBackToFairyWhenPikafishUnavailable(t *testing.T) {
	pikafish := &stubAdapter{src: SourcePikafish, available: false}
	fairy := &stubAdapter{src: SourceFairyStockfish, available: true}
	registry := NewRegistry(pikafish, fairy)

	primary, ok := registry.Primary()
	if !ok {
		t.Fatalf("expected fallback primary, got none")
	}
	if primary.Source() != SourceFairyStockfish {
		t.Errorf("fallback source = %q, want fairy-stockfish", primary.Source())
	}
}

func TestRegistryReturnsEmptyWhenNoEnginesAvailable(t *testing.T) {
	registry := NewRegistry(
		&stubAdapter{src: SourcePikafish, available: false},
		&stubAdapter{src: SourceFairyStockfish, available: false},
	)
	if _, ok := registry.Primary(); ok {
		t.Fatalf("expected no primary, got one")
	}
	if got := registry.Available(); len(got) != 0 {
		t.Errorf("available sources = %v, want empty", got)
	}
}

func TestRegistryGetReturnsAdapterForSource(t *testing.T) {
	registry := NewRegistry(
		&stubAdapter{src: SourcePikafish, available: true},
		&stubAdapter{src: SourceFairyStockfish, available: false},
	)
	// Get returns the registered adapter regardless of availability; callers
	// use Available() to decide whether to actually invoke it.
	if adapter := registry.Get(SourceFairyStockfish); adapter == nil {
		t.Errorf("expected registered fairy adapter, got nil")
	}
	if adapter := registry.Get(SourcePikafish); adapter == nil {
		t.Errorf("expected pikafish adapter, got nil")
	}
	if adapter := registry.Get(Source("nope")); adapter != nil {
		t.Errorf("expected nil for unknown source, got %v", adapter)
	}
}

func TestRegistryAvailableListsLiveSources(t *testing.T) {
	registry := NewRegistry(
		&stubAdapter{src: SourcePikafish, available: true},
		&stubAdapter{src: SourceFairyStockfish, available: true},
	)
	got := registry.Available()
	if len(got) != 2 {
		t.Fatalf("available = %v, want 2 sources", got)
	}
}

func TestPikafishAdapterAvailableRequiresBinary(t *testing.T) {
	adapter := NewPikafishAdapter(Config{Path: ""})
	if adapter.Available() {
		t.Errorf("expected unavailable when path is empty")
	}

	adapter = NewPikafishAdapter(Config{Path: "/definitely/does/not/exist"})
	if adapter.Available() {
		t.Errorf("expected unavailable when path is invalid")
	}
}

func TestPikafishAdapterCapabilities(t *testing.T) {
	adapter := NewPikafishAdapter(Config{Path: "/tmp/whatever"})
	caps := adapter.Capabilities()
	if !caps.MultiPV {
		t.Errorf("expected MultiPV true for pikafish")
	}
	if !caps.SupportsXiangqi {
		t.Errorf("expected SupportsXiangqi true for pikafish")
	}
	if adapter.Source() != SourcePikafish {
		t.Errorf("source = %q, want pikafish", adapter.Source())
	}
}

func TestFairyStockfishAdapterAvailableRequiresBinary(t *testing.T) {
	adapter := NewFairyStockfishAdapter(Config{Path: ""})
	if adapter.Available() {
		t.Errorf("expected unavailable when path is empty")
	}

	adapter = NewFairyStockfishAdapter(Config{Path: "/definitely/does/not/exist"})
	if adapter.Available() {
		t.Errorf("expected unavailable when path is invalid")
	}
}
