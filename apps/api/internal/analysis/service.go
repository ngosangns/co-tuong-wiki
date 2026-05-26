package analysis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrEngineUnavailable = errors.New("strong xiangqi engine is not configured")
	ErrEngineTimeout     = errors.New("engine analysis timed out")
)

type Analyzer interface {
	Analyze(ctx context.Context, request Request) (Response, error)
}

func Analyze(ctx context.Context, request Request) (Response, error) {
	config := ConfigFromEnv()
	if config.Kind != "uci" {
		return Response{}, fmt.Errorf("%w: ENGINE_KIND must be uci", ErrEngineUnavailable)
	}
	if strings.TrimSpace(config.Path) == "" {
		return Response{}, fmt.Errorf("%w: set ENGINE_PATH to a UCI xiangqi engine binary", ErrEngineUnavailable)
	}

	analyzer := NewUCIAnalyzer(config)
	timeout := requestTimeout(request, config)
	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return analyzer.Analyze(timedCtx, request)
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
