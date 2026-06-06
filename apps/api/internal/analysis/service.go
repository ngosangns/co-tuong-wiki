package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrEngineUnavailable = errors.New("strong xiangqi engine is not configured")
	ErrEngineTimeout     = errors.New("engine analysis timed out")
)

type Analyzer interface {
	Analyze(ctx context.Context, request Request) (Response, error)
}

type AnalyzeMeta struct {
	CacheStatus string
}

// Shared engines keep the UCI process warm across HTTP requests; the analyzer itself serializes searches.
var sharedEngines = newSharedEngineState()

func Analyze(ctx context.Context, request Request) (Response, error) {
	response, _, err := AnalyzeWithMeta(ctx, request)
	return response, err
}

func AnalyzeWithMeta(ctx context.Context, request Request) (Response, AnalyzeMeta, error) {
	config := ConfigFromEnv()
	if config.Kind != "uci" {
		return Response{}, AnalyzeMeta{}, fmt.Errorf("%w: ENGINE_KIND must be uci", ErrEngineUnavailable)
	}
	if strings.TrimSpace(config.Path) == "" {
		return Response{}, AnalyzeMeta{}, fmt.Errorf("%w: set ENGINE_PATH to a UCI xiangqi engine binary", ErrEngineUnavailable)
	}

	cacheKey := analysisCacheKey(config, request)
	if response, ok, active, shared := sharedEngines.begin(cacheKey); ok {
		return response, AnalyzeMeta{CacheStatus: "hit"}, nil
	} else if shared {
		response, err := active.wait(ctx)
		if err != nil {
			return Response{}, AnalyzeMeta{CacheStatus: "coalesced"}, err
		}
		return response, AnalyzeMeta{CacheStatus: "coalesced"}, nil
	} else {
		defer func() {
			if recovered := recover(); recovered != nil {
				sharedEngines.finish(cacheKey, active, Response{}, fmt.Errorf("engine analysis panic: %v", recovered))
				panic(recovered)
			}
		}()

		analyzer := sharedEngines.analyzer(config)
		timeout := requestTimeout(request, config)
		timedCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		response, err := analyzer.Analyze(timedCtx, request)
		sharedEngines.finish(cacheKey, active, response, err)
		if err != nil {
			return Response{}, AnalyzeMeta{CacheStatus: "miss"}, err
		}
		return response, AnalyzeMeta{CacheStatus: "miss"}, nil
	}
}

func requestTimeout(request Request, config Config) time.Duration {
	timeMS := request.TimeMS
	if timeMS <= 0 {
		timeMS = config.DefaultTimeMS
	}
	if timeMS > config.MaxTimeMS {
		timeMS = config.MaxTimeMS
	}

	// UCI engines need a little room to initialize and flush bestmove after the requested search window.
	return time.Duration(timeMS)*time.Millisecond + config.TimeoutSlack
}

type sharedEngineState struct {
	mu             sync.Mutex
	config         Config
	activeAnalyzer *PersistentUCIAnalyzer
	cache          *analysisCache
	inFlight       map[string]*analysisCall
}

func newSharedEngineState() *sharedEngineState {
	return &sharedEngineState{
		cache:    newAnalysisCache(256),
		inFlight: map[string]*analysisCall{},
	}
}

func (state *sharedEngineState) analyzer(config Config) *PersistentUCIAnalyzer {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.activeAnalyzer != nil && state.config == config {
		return state.activeAnalyzer
	}

	if state.activeAnalyzer != nil {
		_ = state.activeAnalyzer.Close()
	}
	state.config = config
	state.activeAnalyzer = NewPersistentUCIAnalyzer(config)
	state.cache.clear()
	return state.activeAnalyzer
}

func (state *sharedEngineState) begin(key string) (Response, bool, *analysisCall, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if response, ok := state.cache.get(key); ok {
		return response, true, nil, false
	}
	if active := state.inFlight[key]; active != nil {
		return Response{}, false, active, true
	}

	active := &analysisCall{done: make(chan struct{})}
	state.inFlight[key] = active
	return Response{}, false, active, false
}

func (state *sharedEngineState) finish(key string, active *analysisCall, response Response, err error) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if err == nil {
		state.cache.set(key, response)
	}
	active.response = response
	active.err = err
	close(active.done)
	delete(state.inFlight, key)
}

func (state *sharedEngineState) close() {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.activeAnalyzer != nil {
		_ = state.activeAnalyzer.Close()
	}
	state.activeAnalyzer = nil
	state.cache.clear()
	state.inFlight = map[string]*analysisCall{}
}

type analysisCall struct {
	done     chan struct{}
	response Response
	err      error
}

func (call *analysisCall) wait(ctx context.Context) (Response, error) {
	select {
	case <-call.done:
		return call.response, call.err
	case <-ctx.Done():
		return Response{}, ctx.Err()
	}
}

type analysisCache struct {
	maxEntries int
	entries    map[string]analysisCacheEntry
	order      []string
}

type analysisCacheEntry struct {
	response  Response
	expiresAt time.Time
}

const analysisCacheTTL = 10 * time.Minute

func newAnalysisCache(maxEntries int) *analysisCache {
	return &analysisCache{
		maxEntries: maxEntries,
		entries:    map[string]analysisCacheEntry{},
	}
}

func (cache *analysisCache) get(key string) (Response, bool) {
	entry, ok := cache.entries[key]
	if !ok {
		return Response{}, false
	}
	if time.Now().After(entry.expiresAt) {
		cache.delete(key)
		return Response{}, false
	}
	return entry.response, true
}

func (cache *analysisCache) set(key string, response Response) {
	if _, exists := cache.entries[key]; !exists {
		cache.order = append(cache.order, key)
	}
	cache.entries[key] = analysisCacheEntry{
		response:  response,
		expiresAt: time.Now().Add(analysisCacheTTL),
	}

	for len(cache.order) > cache.maxEntries {
		oldest := cache.order[0]
		cache.order = cache.order[1:]
		delete(cache.entries, oldest)
	}
}

func (cache *analysisCache) clear() {
	cache.entries = map[string]analysisCacheEntry{}
	cache.order = nil
}

func (cache *analysisCache) delete(key string) {
	delete(cache.entries, key)
	nextOrder := cache.order[:0]
	for _, orderedKey := range cache.order {
		if orderedKey != key {
			nextOrder = append(nextOrder, orderedKey)
		}
	}
	cache.order = nextOrder
}

func analysisCacheKey(config Config, request Request) string {
	nextMove := ""
	if request.NextMove != nil {
		nextMove = fmt.Sprintf("%d,%d:%d,%d", request.NextMove.From.File, request.NextMove.From.Rank, request.NextMove.To.File, request.NextMove.To.Rank)
	}
	// The cache key includes search settings because the same FEN can legitimately produce different depth/time results.
	return fmt.Sprintf(
		"%s|%d|%d|%d|%d|%s|%s|%d|%d|%s",
		config.Path,
		config.DefaultTimeMS,
		config.MaxTimeMS,
		config.DefaultDepth,
		config.MaxDepth,
		request.FEN,
		request.SideToMove,
		request.TimeMS,
		request.Depth,
		nextMove,
	)
}
