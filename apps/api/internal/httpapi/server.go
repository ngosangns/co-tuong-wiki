package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"co-tuong-wiki-api/internal/analysis"
	"co-tuong-wiki-api/internal/cacheutil"
	"co-tuong-wiki-api/internal/lessons"
)

type Server struct {
	repository    *lessons.Repository
	mux           *http.ServeMux
	combinedMu    sync.Mutex
	combinedCache *combinedLessonCache
}

type combinedLessonCache struct {
	lesson        lessons.Lesson
	version       cacheutil.DataVersion
	overview      combinedLessonOverview
	overviewBytes []byte
	byLineID      map[string]lessons.Line
	maxMoves      int
	windowBytes   *cacheutil.ByteCache
	inFlight      *cacheutil.Group[[]byte]
}

type combinedLessonOverview struct {
	ID         string                `json:"id"`
	Title      string                `json:"title"`
	Category   string                `json:"category"`
	Difficulty string                `json:"difficulty"`
	InitialFEN string                `json:"initialFen,omitempty"`
	Lines      []combinedLineSummary `json:"lines"`
	Choice     lessons.Choice        `json:"choice"`
}

type combinedLineSummary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	InitialFEN string `json:"initialFen,omitempty"`
	Phase      string `json:"phase"`
	PieceCount int    `json:"pieceCount"`
	MoveCount  int    `json:"moveCount"`
}

type combinedLineMoveWindow struct {
	LineID     string         `json:"lineId"`
	Phase      string         `json:"phase"`
	PieceCount int            `json:"pieceCount"`
	From       int            `json:"from"`
	Moves      []lessons.Move `json:"moves"`
	TotalMoves int            `json:"totalMoves"`
}

type combinedStepMoveWindow struct {
	From  int                      `json:"from"`
	Limit int                      `json:"limit"`
	Lines []combinedLineMoveWindow `json:"lines"`
}

type combinedNextStepsRequest struct {
	Phase      string         `json:"phase"`
	InitialFEN string         `json:"initialFen"`
	From       int            `json:"from"`
	Limit      int            `json:"limit"`
	Prefix     []lessons.Move `json:"prefix"`
}

const defaultXiangqiFEN = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"

func NewServer(repository *lessons.Repository) http.Handler {
	server := &Server{
		repository: repository,
		mux:        http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if isAllowedDevOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-None-Match")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Expose-Headers", "ETag, Last-Modified, X-Cache, X-Data-Version")
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	s.mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) {
		writeCacheableJSONBytes(w, r, s.repository.CategoriesBytes(), s.repository.Version(), "categories", "hit")
	})

	s.mux.HandleFunc("GET /api/lessons", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		body, err := s.repository.ListBytes(query.Get("category"), query.Get("q"), query.Get("difficulty"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not encode lessons")
			return
		}
		writeCacheableJSONBytes(w, r, body, s.repository.Version(), "lessons:"+cacheutil.HashKey(query.Get("category"), query.Get("q"), query.Get("difficulty")), "hit")
	})

	s.mux.HandleFunc("GET /api/combined-lesson", func(w http.ResponseWriter, r *http.Request) {
		combined, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}
		writeCacheableJSONBytes(w, r, combined.overviewBytes, combined.version, "combined-overview", "hit")
	})

	s.mux.HandleFunc("GET /api/combined-lesson/moves", func(w http.ResponseWriter, r *http.Request) {
		combined, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}

		from := queryInt(r, "from", 0, 0, combined.maxMoves)
		limit := queryInt(r, "limit", 1, 1, 8)
		phase := r.URL.Query().Get("phase")
		key := cacheutil.HashKey("combined-moves", phase, strconv.Itoa(from), strconv.Itoa(limit))
		body, cacheState, err := combined.cachedWindowBytes(r.Context(), key, func() (any, error) {
			return combinedMoveWindow(combined.lesson, from, limit, phase), nil
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not encode combined moves")
			return
		}
		writeCacheableJSONBytes(w, r, body, combined.version, "combined-moves:"+key, cacheState)
	})

	s.mux.HandleFunc("POST /api/combined-lesson/next-steps", func(w http.ResponseWriter, r *http.Request) {
		combined, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}

		var request combinedNextStepsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid combined next steps request")
			return
		}

		from := min(max(request.From, 0), combined.maxMoves)
		limit := min(max(request.Limit, 1), 8)
		key := combinedNextStepsCacheKey(request, from, limit)
		body, cacheState, err := combined.cachedWindowBytes(r.Context(), key, func() (any, error) {
			return combinedNextStepWindow(combined.lesson, request.Phase, request.InitialFEN, request.Prefix, from, limit), nil
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not encode combined next steps")
			return
		}
		writeVersionedJSONBytes(w, http.StatusOK, body, combined.version, cacheState)
	})

	s.mux.HandleFunc("GET /api/combined-lesson/lines/{id}/moves", func(w http.ResponseWriter, r *http.Request) {
		combined, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}

		line, ok := combined.byLineID[r.PathValue("id")]
		if !ok {
			writeError(w, http.StatusNotFound, "combined lesson line not found")
			return
		}

		from := queryInt(r, "from", 0, 0, len(line.Moves))
		limit := queryInt(r, "limit", 12, 1, 48)
		to := min(from+limit, len(line.Moves))
		key := cacheutil.HashKey("combined-line-moves", line.ID, strconv.Itoa(from), strconv.Itoa(limit))
		body, cacheState, err := combined.cachedWindowBytes(r.Context(), key, func() (any, error) {
			return combinedLineMoveWindow{
				LineID:     line.ID,
				Phase:      combinedLinePhase(combined.lesson, line),
				PieceCount: effectiveLinePieceCount(combined.lesson, line),
				From:       from,
				Moves:      line.Moves[from:to],
				TotalMoves: len(line.Moves),
			}, nil
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not encode combined line moves")
			return
		}
		writeCacheableJSONBytes(w, r, body, combined.version, "combined-line-moves:"+key, cacheState)
	})

	s.mux.HandleFunc("GET /api/lessons/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		body, ok, err := s.repository.LessonBytes(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not encode lesson")
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "lesson not found")
			return
		}
		writeCacheableJSONBytes(w, r, body, s.repository.Version(), "lesson:"+id, "hit")
	})

	s.mux.HandleFunc("POST /api/analyze", func(w http.ResponseWriter, r *http.Request) {
		var request analysis.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid analyze request")
			return
		}
		if strings.TrimSpace(request.FEN) == "" {
			writeError(w, http.StatusBadRequest, "fen is required")
			return
		}
		response, meta, err := analysis.AnalyzeWithMeta(r.Context(), request)
		if err != nil {
			status, message := analyzeErrorResponse(err)
			writeError(w, status, message)
			return
		}
		if meta.CacheStatus != "" {
			w.Header().Set("X-Cache", meta.CacheStatus)
		}
		writeJSON(w, http.StatusOK, response)
	})
}

func (s *Server) loadCombinedLesson() (*combinedLessonCache, error) {
	s.combinedMu.Lock()
	defer s.combinedMu.Unlock()

	if s.combinedCache != nil {
		return s.combinedCache, nil
	}

	lesson, version, err := lessons.LoadLessonWithVersion(lessons.DefaultCombinedDataPath())
	if err != nil {
		return nil, err
	}
	cache, err := newCombinedLessonCache(lesson, version)
	if err != nil {
		return nil, err
	}
	s.combinedCache = cache
	return s.combinedCache, nil
}

func newCombinedLessonCache(lesson lessons.Lesson, version cacheutil.DataVersion) (*combinedLessonCache, error) {
	overview := combinedOverview(lesson)
	overviewBytes, err := json.Marshal(overview)
	if err != nil {
		return nil, err
	}

	return &combinedLessonCache{
		lesson:        lesson,
		version:       version,
		overview:      overview,
		overviewBytes: overviewBytes,
		byLineID:      combinedLineIndex(lesson),
		maxMoves:      maxCombinedMoveCount(lesson),
		windowBytes:   cacheutil.NewByteCache(512),
		inFlight:      cacheutil.NewGroup[[]byte](),
	}, nil
}

func (cache *combinedLessonCache) cachedWindowBytes(ctx context.Context, key string, build func() (any, error)) ([]byte, string, error) {
	if body, ok := cache.windowBytes.Get(key); ok {
		return body, "hit", nil
	}

	body, shared, err := cache.inFlight.Do(ctx, key, func() ([]byte, error) {
		if body, ok := cache.windowBytes.Get(key); ok {
			return body, nil
		}

		value, err := build()
		if err != nil {
			return nil, err
		}
		body, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		cache.windowBytes.Set(key, body)
		return body, nil
	})
	if err != nil {
		return nil, "miss", err
	}
	if shared {
		return body, "coalesced", nil
	}

	return body, "miss", nil
}

func combinedLineIndex(lesson lessons.Lesson) map[string]lessons.Line {
	byID := make(map[string]lessons.Line, len(lesson.Lines))
	for _, line := range lesson.Lines {
		byID[line.ID] = line
	}
	return byID
}

func combinedOverview(lesson lessons.Lesson) combinedLessonOverview {
	lines := make([]combinedLineSummary, 0, len(lesson.Lines))
	for _, line := range lesson.Lines {
		lines = append(lines, combinedLineSummary{
			ID:         line.ID,
			Title:      line.Title,
			InitialFEN: line.InitialFEN,
			Phase:      combinedLinePhase(lesson, line),
			PieceCount: effectiveLinePieceCount(lesson, line),
			MoveCount:  len(line.Moves),
		})
	}

	return combinedLessonOverview{
		ID:         lesson.ID,
		Title:      lesson.Title,
		Category:   lesson.Category,
		Difficulty: lesson.Difficulty,
		InitialFEN: effectiveLessonInitialFEN(lesson),
		Lines:      lines,
		Choice:     lesson.Choice,
	}
}

func maxCombinedMoveCount(lesson lessons.Lesson) int {
	maxMoves := 0
	for _, line := range lesson.Lines {
		maxMoves = max(maxMoves, len(line.Moves))
	}
	return maxMoves
}

func combinedNextStepWindow(lesson lessons.Lesson, phase string, initialFEN string, prefix []lessons.Move, from int, limit int) combinedStepMoveWindow {
	lines := make([]combinedLineMoveWindow, 0, len(lesson.Lines))
	startFEN := strings.TrimSpace(initialFEN)

	for _, line := range lesson.Lines {
		linePhase := combinedLinePhase(lesson, line)
		if phase != "" && phase != linePhase {
			continue
		}
		if startFEN != "" && strings.TrimSpace(effectiveLineInitialFEN(lesson, line)) != startFEN {
			continue
		}
		// The prefix identifies the selected graph node, so the response only hydrates sibling continuations.
		if !lineHasMovePrefix(line, prefix) {
			continue
		}

		to := min(from+limit, len(line.Moves))
		moves := []lessons.Move{}
		if from < to {
			moves = line.Moves[from:to]
		}
		lines = append(lines, combinedLineMoveWindow{
			LineID:     line.ID,
			Phase:      linePhase,
			PieceCount: effectiveLinePieceCount(lesson, line),
			From:       from,
			Moves:      moves,
			TotalMoves: len(line.Moves),
		})
	}

	return combinedStepMoveWindow{
		From:  from,
		Limit: limit,
		Lines: lines,
	}
}

func combinedNextStepsCacheKey(request combinedNextStepsRequest, from int, limit int) string {
	return cacheutil.HashKey(
		"combined-next-steps",
		request.Phase,
		request.InitialFEN,
		strconv.Itoa(from),
		strconv.Itoa(limit),
		movePrefixSignature(request.Prefix),
	)
}

func movePrefixSignature(prefix []lessons.Move) string {
	var builder strings.Builder
	for _, move := range prefix {
		builder.WriteString(move.Side)
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(move.From.File))
		builder.WriteByte(',')
		builder.WriteString(strconv.Itoa(move.From.Rank))
		builder.WriteString(">")
		builder.WriteString(strconv.Itoa(move.To.File))
		builder.WriteByte(',')
		builder.WriteString(strconv.Itoa(move.To.Rank))
		builder.WriteByte(';')
	}
	return builder.String()
}

func lineHasMovePrefix(line lessons.Line, prefix []lessons.Move) bool {
	if len(prefix) > len(line.Moves) {
		return false
	}
	for index, move := range prefix {
		if !sameMoveStep(line.Moves[index], move) {
			return false
		}
	}
	return true
}

func sameMoveStep(left lessons.Move, right lessons.Move) bool {
	return left.Side == right.Side &&
		left.From == right.From &&
		left.To == right.To
}

func combinedMoveWindow(lesson lessons.Lesson, from int, limit int, phase string) combinedStepMoveWindow {
	lines := make([]combinedLineMoveWindow, 0, len(lesson.Lines))
	for _, line := range lesson.Lines {
		linePhase := combinedLinePhase(lesson, line)
		if phase != "" && phase != linePhase {
			continue
		}
		to := min(from+limit, len(line.Moves))
		moves := []lessons.Move{}
		if from < to {
			moves = line.Moves[from:to]
		}
		lines = append(lines, combinedLineMoveWindow{
			LineID:     line.ID,
			Phase:      linePhase,
			PieceCount: effectiveLinePieceCount(lesson, line),
			From:       from,
			Moves:      moves,
			TotalMoves: len(line.Moves),
		})
	}

	return combinedStepMoveWindow{
		From:  from,
		Limit: limit,
		Lines: lines,
	}
}

func combinedLinePhase(lesson lessons.Lesson, line lessons.Line) string {
	pieceCount := effectiveLinePieceCount(lesson, line)
	switch {
	case pieceCount >= 28:
		return "opening"
	case pieceCount >= 14:
		return "middlegame"
	default:
		return "endgame"
	}
}

func effectiveLinePieceCount(lesson lessons.Lesson, line lessons.Line) int {
	return countFENPieces(effectiveLineInitialFEN(lesson, line))
}

func effectiveLineInitialFEN(lesson lessons.Lesson, line lessons.Line) string {
	fen := strings.TrimSpace(line.InitialFEN)
	if fen != "" {
		return fen
	}
	return effectiveLessonInitialFEN(lesson)
}

func effectiveLessonInitialFEN(lesson lessons.Lesson) string {
	fen := strings.TrimSpace(lesson.InitialFEN)
	if fen != "" {
		return fen
	}
	// Opening lines without a custom FEN still need an explicit root state in the combined graph.
	return defaultXiangqiFEN
}

func countFENPieces(fen string) int {
	placement := strings.Fields(fen)
	if len(placement) == 0 {
		return 32
	}

	count := 0
	for _, symbol := range placement[0] {
		if symbol >= 'A' && symbol <= 'Z' || symbol >= 'a' && symbol <= 'z' {
			count++
		}
	}
	return count
}

func queryInt(r *http.Request, name string, fallback int, minValue int, maxValue int) int {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return min(max(parsed, minValue), maxValue)
}

func analyzeErrorResponse(err error) (int, string) {
	if errors.Is(err, analysis.ErrEngineUnavailable) {
		return http.StatusServiceUnavailable, "Engine phân tích chưa được cấu hình."
	}
	if errors.Is(err, analysis.ErrEngineTimeout) {
		return http.StatusGatewayTimeout, "Engine phân tích quá lâu, vui lòng thử lại với thời gian hoặc độ sâu thấp hơn."
	}
	return http.StatusBadGateway, "Engine không thể phân tích vị trí hiện tại."
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeCacheableJSONBytes(w http.ResponseWriter, r *http.Request, body []byte, version cacheutil.DataVersion, scope string, cacheStatus string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	setVersionHeaders(w, version)
	if cacheStatus != "" {
		w.Header().Set("X-Cache", cacheStatus)
	}

	if etag := version.ETag(scope); etag != "" {
		w.Header().Set("ETag", etag)
		if matchesETag(r.Header.Get("If-None-Match"), etag) {
			w.Header().Set("X-Cache", "revalidate")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func writeVersionedJSONBytes(w http.ResponseWriter, status int, body []byte, version cacheutil.DataVersion, cacheStatus string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	setVersionHeaders(w, version)
	if cacheStatus != "" {
		w.Header().Set("X-Cache", cacheStatus)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func setVersionHeaders(w http.ResponseWriter, version cacheutil.DataVersion) {
	if version.Value == "" {
		return
	}
	w.Header().Set("X-Data-Version", version.Value)
	if !version.LastModified.IsZero() {
		w.Header().Set("Last-Modified", version.LastModified.Format(http.TimeFormat))
	}
}

func matchesETag(header string, etag string) bool {
	for _, value := range strings.Split(header, ",") {
		value = strings.TrimSpace(value)
		if value == "*" || value == etag {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func isAllowedDevOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:")
}
