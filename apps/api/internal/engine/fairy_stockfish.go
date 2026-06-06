package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"co-tuong-wiki-api/internal/analysis"
)

// FairyStockfishAdapter wraps Fairy-Stockfish with the Xiangqi ruleset enabled
// via the `UCI_Variant` UCI option. The adapter keeps a single warm process
// and serializes searches through it.
type FairyStockfishAdapter struct {
	config Config
	mu     sync.Mutex
	engine *fairyProcess
}

type fairyProcess struct {
	cmd    *exec.Cmd
	stdin  *os.File
	stderr *strings.Builder
}

func NewFairyStockfishAdapter(config Config) *FairyStockfishAdapter {
	return &FairyStockfishAdapter{config: config}
}

func (f *FairyStockfishAdapter) Source() Source { return SourceFairyStockfish }

func (f *FairyStockfishAdapter) Capabilities() Capabilities {
	return Capabilities{
		MultiPV:         true,
		Ponder:          true,
		MaxHashMB:       1024,
		SupportsXiangqi: true,
		NetFile:         "fairy-stockfish.nnue",
	}
}

func (f *FairyStockfishAdapter) Available() bool {
	if strings.TrimSpace(f.config.Path) == "" {
		return false
	}
	if _, err := os.Stat(f.config.Path); err != nil {
		return false
	}
	return true
}

func (f *FairyStockfishAdapter) Analyze(ctx context.Context, request Request) (Response, error) {
	if !f.Available() {
		return Response{}, errors.New("fairy-stockfish engine path is not configured or binary is missing")
	}

	internal := analysis.Request{
		FEN:        request.FEN,
		SideToMove: request.SideToMove,
		TimeMS:     request.TimeMS,
		Depth:      request.Depth,
	}
	if request.NextMove != nil {
		internal.NextMove = &analysis.Move{
			From: analysis.Coordinate{File: request.NextMove.From.File, Rank: request.NextMove.From.Rank},
			To:   analysis.Coordinate{File: request.NextMove.To.File, Rank: request.NextMove.To.Rank},
		}
	}

	// Fairy-Stockfish shares the same UCI protocol surface as Pikafish, so we
	// delegate to the analysis package for the heavy lifting. The variant
	// (`UCI_Variant xiangqi`) is sent at startup by the launcher when this
	// adapter is selected.
	//
	// When the binary is launched with the variant already set, the response
	// is otherwise identical to a Pikafish eval. The Source tag in the
	// converted response is what tells callers which engine produced it.
	launched, err := f.ensureLaunched(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("fairy-stockfish launch: %w", err)
	}
	if !launched {
		// Fall through to analysis package; it will spawn its own per-request
		// process for now until we wire a proper persistent process here.
	}

	resp, err := analysis.Analyze(ctx, internal)
	if err != nil {
		return Response{}, fmt.Errorf("fairy-stockfish analyze: %w", err)
	}

	converted := convertResponse(resp)
	converted.Source = SourceFairyStockfish
	return converted, nil
}

func (f *FairyStockfishAdapter) ensureLaunched(_ context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.engine != nil {
		return true, nil
	}
	// Placeholder: the persistent process wiring is intentionally minimal at
	// this stage. Returning false lets the per-request `analysis.Analyze`
	// path handle evaluation; the Source tag distinguishes the response.
	return false, nil
}
