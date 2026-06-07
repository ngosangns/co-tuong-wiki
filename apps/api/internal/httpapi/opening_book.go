package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"co-tuong-wiki-api/internal/cacheutil"
)

// openingBookEntry is the on-disk shape of a single position record.
type openingBookEntry struct {
	Moves []openingBookMove `json:"moves"`
	Visits int              `json:"visits"`
}

type openingBookMove struct {
	Move       string  `json:"move"`
	Notation   string  `json:"notation"`
	Name       string  `json:"name"`
	Frequency  int     `json:"frequency"`
	Popularity float64 `json:"popularity"`
}

type openingBookFile struct {
	Version   int                        `json:"version"`
	Depth     int                        `json:"depth"`
	Source    string                     `json:"source"`
	Stats     map[string]any             `json:"stats"`
	Positions map[string]openingBookEntry `json:"positions"`
}

var (
	openingBookOnce   sync.Once
	openingBookValue  *openingBookFile
	openingBookErr    error
	openingBookLoaded time.Time
)

const openingBookTTL = 10 * time.Minute

// loadOpeningBook reads the curated opening book artifact on first use and
// caches it in memory. The artifact lives in apps/api/data/book/opening-book.json
// next to the lesson catalog so the build_opening_book.py script can refresh it
// without restarting the API.
func loadOpeningBook() (*openingBookFile, error) {
	openingBookOnce.Do(func() {
		path := openingBookPath()
		data, err := os.ReadFile(path)
		if err != nil {
			openingBookErr = err
			return
		}
		var book openingBookFile
		if err := json.Unmarshal(data, &book); err != nil {
			openingBookErr = err
			return
		}
		openingBookValue = &book
		openingBookLoaded = time.Now()
	})
	if openingBookErr != nil {
		return nil, openingBookErr
	}
	if time.Since(openingBookLoaded) > openingBookTTL {
		// The book is small (~500KB); we just keep the in-memory copy for
		// the process lifetime and re-read on demand if the file changed.
		path := openingBookPath()
		data, err := os.ReadFile(path)
		if err != nil {
			return openingBookValue, nil
		}
		var book openingBookFile
		if err := json.Unmarshal(data, &book); err == nil {
			openingBookValue = &book
			openingBookLoaded = time.Now()
		}
	}
	return openingBookValue, nil
}

func openingBookPath() string {
	if fromEnv := os.Getenv("OPENING_BOOK_FILE"); strings.TrimSpace(fromEnv) != "" {
		return fromEnv
	}
	return defaultOpeningBookPath()
}

func defaultOpeningBookPath() string {
	candidates := []string{
		"apps/api/data/book/opening-book.json",
		"data/book/opening-book.json",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

type openingAPIResponse struct {
	Available bool                  `json:"available"`
	FEN       string                `json:"fen"`
	Visits    int                   `json:"visits"`
	Moves     []openingAPIMove      `json:"moves"`
	Stats     map[string]any        `json:"stats,omitempty"`
}

type openingAPIMove struct {
	Move       string  `json:"move"`
	Notation   string  `json:"notation"`
	Name       string  `json:"name"`
	Frequency  int     `json:"frequency"`
	Popularity float64 `json:"popularity"`
}

func (s *Server) handleOpeningLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	fen := strings.TrimSpace(r.URL.Query().Get("fen"))
	if fen == "" {
		writeError(w, http.StatusBadRequest, "fen query parameter is required")
		return
	}

	book, err := loadOpeningBook()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Opening book chưa được build. Hãy chạy scripts/build_opening_book.py.")
		return
	}

	entry, ok := book.Positions[fen]
	if !ok {
		// Empty payload is a valid 200 response: the book has no data for
		// this position (e.g. midgame or non-standard FEN).
		writeJSON(w, http.StatusOK, openingAPIResponse{
			Available: true,
			FEN:       fen,
			Visits:    0,
			Moves:     []openingAPIMove{},
			Stats:     book.Stats,
		})
		return
	}

	moves := make([]openingAPIMove, 0, len(entry.Moves))
	for _, m := range entry.Moves {
		moves = append(moves, openingAPIMove{
			Move:       m.Move,
			Notation:   m.Notation,
			Name:       m.Name,
			Frequency:  m.Frequency,
			Popularity: m.Popularity,
		})
	}
	etag := openingETag(fen, entry)
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("X-Cache", "hit")
	writeJSON(w, http.StatusOK, openingAPIResponse{
		Available: true,
		FEN:       fen,
		Visits:    entry.Visits,
		Moves:     moves,
		Stats:     book.Stats,
	})
}

func openingETag(fen string, entry openingBookEntry) string {
	sum := sha256.Sum256([]byte(fen + "|" + openingMovesSignature(entry)))
	hash := cacheutil.HashKey("opening", hex.EncodeToString(sum[:8]))
	return `"` + hash + `"`
}

func openingMovesSignature(entry openingBookEntry) string {
	parts := make([]string, 0, len(entry.Moves))
	for _, m := range entry.Moves {
		parts = append(parts, m.Move+":"+m.Notation+":"+itoa(m.Frequency))
	}
	return strings.Join(parts, "|")
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
