package httpapi

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"co-tuong-wiki-api/internal/cacheutil"
	"co-tuong-wiki-api/internal/engine"
)

type lineEvaluationPly struct {
	Ply        int    `json:"ply"`
	FEN        string `json:"fen"`
	SideToMove string `json:"sideToMove"`
	NextMove   *struct {
		Side string `json:"side"`
		From struct {
			File int `json:"file"`
			Rank int `json:"rank"`
		} `json:"from"`
		To struct {
			File int `json:"file"`
			Rank int `json:"rank"`
		} `json:"to"`
	} `json:"nextMove,omitempty"`
}

type lineEvaluationRequest struct {
	OpeningFen string               `json:"openingFen"`
	Engine     string               `json:"engine,omitempty"`
	Plies      []lineEvaluationPly  `json:"plies"`
}

type lineEvaluationPlyResult struct {
	Ply            int      `json:"ply"`
	FEN            string   `json:"fen"`
	SideToMove     string   `json:"sideToMove"`
	Score          int      `json:"cp"`
	Perspective    string   `json:"perspective"`
	BestMove       string   `json:"bestMove,omitempty"`
	BestScore      *int     `json:"bestScore,omitempty"`
	PV             []string `json:"pv"`
	Depth          int      `json:"depth"`
	Status         string   `json:"status"`
	Error          string   `json:"error,omitempty"`
	CpLoss         *int     `json:"cpLoss,omitempty"`
	Classification string   `json:"classification,omitempty"`
}

type lineEvaluationResponse struct {
	Engine   engine.Source             `json:"engine"`
	Plies    []lineEvaluationPlyResult `json:"plies"`
	CacheKey string                  `json:"-"`
}

const lineEvaluationCacheTTL = time.Hour

var (
	lineEvaluationCacheMu sync.Mutex
	lineEvaluationCache   = newLineEvaluationByteCache(256)
)

func newLineEvaluationByteCache(maxEntries int) *cacheutil.ByteCache {
	return cacheutil.NewByteCache(maxEntries)
}

func (s *Server) handleLineEvaluation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var request lineEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid line evaluation request")
		return
	}
	if len(request.Plies) == 0 {
		writeError(w, http.StatusBadRequest, "plies array is required")
		return
	}
	if len(request.Plies) > 64 {
		writeError(w, http.StatusBadRequest, "too many plies (max 64 per request)")
		return
	}

	engineName := request.Engine
	if engineName == "" {
		engineName = string(engine.SourcePikafish)
	}

	adapter := s.engines.Get(engine.Source(engineName))
	if adapter == nil || !adapter.Available() {
		writeError(w, http.StatusServiceUnavailable, "Engine không khả dụng.")
		return
	}

	cacheKey := buildLineEvaluationCacheKey(engineName, request.OpeningFen, request.Plies)
	if body, ok := lineEvaluationCache.Get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Cache", "hit")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}

	plies := make([]lineEvaluationPlyResult, 0, len(request.Plies))
	for i, ply := range request.Plies {
		engineRequest := engine.Request{
			FEN:        ply.FEN,
			SideToMove: ply.SideToMove,
		}
		if ply.NextMove != nil {
			engineRequest.NextMove = &engine.Move{
				From: engine.Coordinate{File: ply.NextMove.From.File, Rank: ply.NextMove.From.Rank},
				To:   engine.Coordinate{File: ply.NextMove.To.File, Rank: ply.NextMove.To.Rank},
			}
		}
		response, err := adapter.Analyze(r.Context(), engineRequest)
		result := lineEvaluationPlyResult{
			Ply:        ply.Ply,
			FEN:        ply.FEN,
			SideToMove: ply.SideToMove,
			Status:     "ok",
		}
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
		} else {
			result.Score = response.Score.CP
			result.Perspective = response.Score.Perspective
			if response.BestMove != nil {
				result.BestMove = response.BestMove.Notation
				if response.BestMove.Score != nil {
					score := *response.BestMove.Score
					result.BestScore = &score
				}
			}
			for _, move := range response.PrincipalVariation {
				result.PV = append(result.PV, move.Notation)
			}
			result.Depth = response.Depth

			// The classification is for the move that was played to reach
			// plies[i+1], so it needs the score at this ply (ply N, before the
			// move) and the score at the next ply (ply N+1, after the move).
			// The current bestScore is from this ply's analysis, so we defer
			// computing cpLoss until the next iteration when we know both
			// sides of the comparison.
			if ply.NextMove != nil && i+1 < len(request.Plies) {
				nextResult := lineEvaluationPlyResult{}
				// We'll fill nextResult in the next iteration; the final
				// classification is attached to plies[i+1] (the position
				// AFTER the move).
				_ = nextResult
			}
		}
		plies = append(plies, result)
	}

	// Now walk the plies in pairs to compute cpLoss + classification for the
	// move that transitions from plies[i] to plies[i+1]. The result is
	// attached to plies[i+1] because it describes the move that landed there.
	for i := 1; i < len(plies); i++ {
		prev := plies[i-1]
		current := plies[i]
		if current.Status != "ok" || prev.Status != "ok" {
			continue
		}
		bestScore := 0
		hasBest := prev.BestScore != nil
		if hasBest {
			bestScore = *prev.BestScore
		}
		loss := cpLoss(bestScore, current.Score)
		plies[i].CpLoss = &loss
		if !hasBest {
			plies[i].Classification = "unknown"
		} else {
			plies[i].Classification = string(classifyMove(loss))
		}
	}

	payload := lineEvaluationResponse{Engine: engine.Source(engineName), Plies: plies}
	body, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not encode line evaluation")
		return
	}
	lineEvaluationCache.Set(cacheKey, body)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Cache", "miss")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func buildLineEvaluationCacheKey(engine string, openingFen string, plies []lineEvaluationPly) string {
	key := cacheutil.HashKey("line-eval", engine, openingFen)
	for _, ply := range plies {
		key = cacheutil.HashKey(key, ply.FEN, ply.SideToMove)
	}
	return key
}
