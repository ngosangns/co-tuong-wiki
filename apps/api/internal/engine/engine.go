// Package engine defines a pluggable chess-engine adapter contract and a registry
// that picks the right implementation based on configuration.
//
// The legacy `analysis` package remains the canonical Pikafish implementation and
// is reused as the primary engine. The Fairy-Stockfish adapter lives alongside it.
package engine

import "context"

// Source identifies which engine produced an analysis response.
type Source string

const (
	SourcePikafish         Source = "pikafish"
	SourceFairyStockfish   Source = "fairy-stockfish"
	SourceUnconfigured     Source = "unconfigured"
)

// Coordinate mirrors the on-wire position format used by every engine adapter.
type Coordinate struct {
	File int `json:"file"`
	Rank int `json:"rank"`
}

// Move is the engine-neutral move description returned in responses.
type Move struct {
	From     Coordinate `json:"from"`
	To       Coordinate `json:"to"`
	Notation string     `json:"notation,omitempty"`
	Score    *int       `json:"score,omitempty"`
}

// Request is the engine-neutral analysis request payload.
type Request struct {
	FEN        string `json:"fen"`
	SideToMove string `json:"sideToMove"`
	NextMove   *Move  `json:"nextMove,omitempty"`
	TimeMS     int    `json:"timeMs,omitempty"`
	Depth      int    `json:"depth,omitempty"`
}

// Score is the engine score normalized to a single side's perspective.
type Score struct {
	CP          int    `json:"cp"`
	Perspective string `json:"perspective"`
}

// Response is the engine-neutral analysis response payload.
type Response struct {
	FEN                string `json:"fen"`
	SideToMove         string `json:"sideToMove"`
	Score              Score  `json:"score"`
	BestMove           *Move  `json:"bestMove,omitempty"`
	PrincipalVariation []Move `json:"principalVariation"`
	Depth              int    `json:"depth"`
	Source             Source `json:"source"`
	Message            string `json:"message,omitempty"`
}

// Capabilities describes the optional UCI features the engine supports so the
// HTTP layer can advertise them in the future.
type Capabilities struct {
	MultiPV         bool
	Ponder          bool
	MaxHashMB       int
	SupportsXiangqi bool
	BuiltAt         string
	NetFile         string
}

// Config carries the per-engine binary path and search parameters.
type Config struct {
	Kind          string
	Path          string
	DefaultTimeMS int
	MaxTimeMS     int
	DefaultDepth  int
	MaxDepth      int
	TimeoutSlack  int // milliseconds
}

// Engine is the contract every engine adapter must implement.
type Engine interface {
	Source() Source
	Capabilities() Capabilities
	Available() bool
	Analyze(ctx context.Context, request Request) (Response, error)
}

// Registry resolves engine names to concrete adapters, with memoized config
// lookups so successive Analyze calls reuse warm processes.
type Registry struct {
	adapters map[Source]Engine
}

func NewRegistry(adapters ...Engine) *Registry {
	registry := &Registry{adapters: make(map[Source]Engine, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		registry.adapters[adapter.Source()] = adapter
	}
	return registry
}

// Primary returns the configured primary engine, falling back to the first
// available adapter. The second return value is false when nothing usable is
// registered, which the HTTP layer can translate to 503.
func (r *Registry) Primary() (Engine, bool) {
	if adapter, ok := r.adapters[SourcePikafish]; ok && adapter.Available() {
		return adapter, true
	}
	for _, adapter := range r.adapters {
		if adapter.Available() {
			return adapter, true
		}
	}
	return nil, false
}

// Get returns the engine for a given source, or nil if unavailable.
func (r *Registry) Get(source Source) Engine {
	if adapter, ok := r.adapters[source]; ok {
		return adapter
	}
	return nil
}

// Available reports the list of sources that have a working adapter.
func (r *Registry) Available() []Source {
	sources := make([]Source, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		if adapter.Available() {
			sources = append(sources, adapter.Source())
		}
	}
	return sources
}
