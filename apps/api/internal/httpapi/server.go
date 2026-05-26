package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"co-tuong-wiki-api/internal/analysis"
	"co-tuong-wiki-api/internal/lessons"
)

type Server struct {
	repository *lessons.Repository
	mux        *http.ServeMux
}

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
