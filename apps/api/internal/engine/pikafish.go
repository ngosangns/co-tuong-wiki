package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"co-tuong-wiki-api/internal/analysis"
)

// PikafishAdapter is the engine.Engine wrapper around the existing analysis
// package, which already implements the long-lived UCI process + caching.
type PikafishAdapter struct {
	config Config
}

// NewPikafishAdapter builds the adapter from the given config. The adapter
// reports unavailable when no binary path is configured.
func NewPikafishAdapter(config Config) *PikafishAdapter {
	return &PikafishAdapter{config: config}
}

func (p *PikafishAdapter) Source() Source { return SourcePikafish }

func (p *PikafishAdapter) Capabilities() Capabilities {
	return Capabilities{
		MultiPV:         true,
		Ponder:          true,
		MaxHashMB:       1024,
		SupportsXiangqi: true,
		NetFile:         "pikafish.nnue",
	}
}

func (p *PikafishAdapter) Available() bool {
	if strings.TrimSpace(p.config.Path) == "" {
		return false
	}
	if _, err := os.Stat(p.config.Path); err != nil {
		return false
	}
	return true
}

func (p *PikafishAdapter) Analyze(ctx context.Context, request Request) (Response, error) {
	if !p.Available() {
		return Response{}, errors.New("pikafish engine path is not configured or binary is missing")
	}

	translated := analysis.Request{
		FEN:        request.FEN,
		SideToMove: request.SideToMove,
		TimeMS:     request.TimeMS,
		Depth:      request.Depth,
	}
	if request.NextMove != nil {
		translated.NextMove = &analysis.Move{
			From: analysis.Coordinate{File: request.NextMove.From.File, Rank: request.NextMove.From.Rank},
			To:   analysis.Coordinate{File: request.NextMove.To.File, Rank: request.NextMove.To.Rank},
		}
	}

	internal, err := analysis.Analyze(ctx, translated)
	if err != nil {
		return Response{}, fmt.Errorf("pikafish analyze: %w", err)
	}

	converted := convertResponse(internal)
	converted.Source = SourcePikafish
	return converted, nil
}

func convertResponse(internal analysis.Response) Response {
	out := Response{
		FEN:        internal.FEN,
		SideToMove: internal.SideToMove,
		Score: Score{
			CP:          internal.Score.CP,
			Perspective: internal.Score.Perspective,
		},
		Depth:   internal.Depth,
		Message: internal.Message,
	}
	if internal.BestMove != nil {
		out.BestMove = &Move{
			From:     Coordinate{File: internal.BestMove.From.File, Rank: internal.BestMove.From.Rank},
			To:       Coordinate{File: internal.BestMove.To.File, Rank: internal.BestMove.To.Rank},
			Notation: internal.BestMove.Notation,
			Score:    internal.BestMove.Score,
		}
	}
	for _, m := range internal.PrincipalVariation {
		out.PrincipalVariation = append(out.PrincipalVariation, Move{
			From:     Coordinate{File: m.From.File, Rank: m.From.Rank},
			To:       Coordinate{File: m.To.File, Rank: m.To.Rank},
			Notation: m.Notation,
		})
	}
	return out
}
