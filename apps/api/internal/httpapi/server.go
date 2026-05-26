package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"co-tuong-wiki-api/internal/analysis"
	"co-tuong-wiki-api/internal/lessons"
)

type Server struct {
	repository     *lessons.Repository
	mux            *http.ServeMux
	combinedMu     sync.Mutex
	combinedLesson lessons.Lesson
	combinedLoaded bool
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
		writeJSON(w, http.StatusOK, s.repository.Categories())
	})

	s.mux.HandleFunc("GET /api/lessons", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		writeJSON(w, http.StatusOK, s.repository.List(query.Get("category"), query.Get("q"), query.Get("difficulty")))
	})

	s.mux.HandleFunc("GET /api/combined-lesson", func(w http.ResponseWriter, r *http.Request) {
		lesson, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}
		writeJSON(w, http.StatusOK, combinedOverview(lesson))
	})

	s.mux.HandleFunc("GET /api/combined-lesson/moves", func(w http.ResponseWriter, r *http.Request) {
		lesson, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}

		maxMoves := maxCombinedMoveCount(lesson)
		from := queryInt(r, "from", 0, 0, maxMoves)
		limit := queryInt(r, "limit", 1, 1, 8)
		writeJSON(w, http.StatusOK, combinedMoveWindow(lesson, from, limit, r.URL.Query().Get("phase")))
	})

	s.mux.HandleFunc("POST /api/combined-lesson/next-steps", func(w http.ResponseWriter, r *http.Request) {
		lesson, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}

		var request combinedNextStepsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid combined next steps request")
			return
		}

		from := min(max(request.From, 0), maxCombinedMoveCount(lesson))
		limit := min(max(request.Limit, 1), 8)
		writeJSON(w, http.StatusOK, combinedNextStepWindow(lesson, request.Phase, request.InitialFEN, request.Prefix, from, limit))
	})

	s.mux.HandleFunc("GET /api/combined-lesson/lines/{id}/moves", func(w http.ResponseWriter, r *http.Request) {
		lesson, err := s.loadCombinedLesson()
		if err != nil {
			writeError(w, http.StatusNotFound, "combined lesson has not been built")
			return
		}

		line, ok := findCombinedLine(lesson, r.PathValue("id"))
		if !ok {
			writeError(w, http.StatusNotFound, "combined lesson line not found")
			return
		}

		from := queryInt(r, "from", 0, 0, len(line.Moves))
		limit := queryInt(r, "limit", 12, 1, 48)
		to := min(from+limit, len(line.Moves))
		writeJSON(w, http.StatusOK, combinedLineMoveWindow{
			LineID:     line.ID,
			Phase:      combinedLinePhase(lesson, line),
			PieceCount: effectiveLinePieceCount(lesson, line),
			From:       from,
			Moves:      line.Moves[from:to],
			TotalMoves: len(line.Moves),
		})
	})

	s.mux.HandleFunc("GET /api/lessons/{id}", func(w http.ResponseWriter, r *http.Request) {
		lesson, ok := s.repository.Get(r.PathValue("id"))
		if !ok {
			writeError(w, http.StatusNotFound, "lesson not found")
			return
		}
		writeJSON(w, http.StatusOK, lesson)
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
		response, err := analysis.Analyze(r.Context(), request)
		if err != nil {
			status, message := analyzeErrorResponse(err)
			writeError(w, status, message)
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
}

func (s *Server) loadCombinedLesson() (lessons.Lesson, error) {
	s.combinedMu.Lock()
	defer s.combinedMu.Unlock()

	if s.combinedLoaded {
		return s.combinedLesson, nil
	}

	lesson, err := lessons.LoadLesson(lessons.DefaultCombinedDataPath())
	if err != nil {
		return lessons.Lesson{}, err
	}
	s.combinedLesson = lesson
	s.combinedLoaded = true
	return lesson, nil
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

func findCombinedLine(lesson lessons.Lesson, lineID string) (lessons.Line, bool) {
	for _, line := range lesson.Lines {
		if line.ID == lineID {
			return line, true
		}
	}
	return lessons.Line{}, false
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

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func isAllowedDevOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:")
}
