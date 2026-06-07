package lessons

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"co-tuong-wiki-api/internal/cacheutil"
)

type Repository struct {
	lessons        []Lesson
	byID           map[string]Lesson
	summaries      []Summary
	categories     []string
	version        cacheutil.DataVersion
	categoriesJSON []byte

	mu         sync.RWMutex
	listJSON   map[string][]byte
	lessonJSON map[string][]byte
}

func DefaultDataPath() string {
	if fromEnv := os.Getenv("LESSONS_FILE"); fromEnv != "" {
		return fromEnv
	}

	return defaultLessonsDataPath("lessons.json")
}

func DefaultCombinedDataPath() string {
	if fromEnv := os.Getenv("COMBINED_LESSON_FILE"); fromEnv != "" {
		return fromEnv
	}

	return defaultLessonsDataPath("combined-lesson.json")
}

func defaultLessonsDataPath(fileName string) string {
	_, sourceFile, _, _ := runtime.Caller(0)
	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	candidates := []string{
		filepath.Join(apiRoot, "data", "lessons", fileName),
		filepath.Join("apps", "api", "data", "lessons", fileName),
		filepath.Join("data", "lessons", fileName),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return candidates[0]
}

func LoadLesson(path string) (Lesson, error) {
	lesson, _, err := LoadLessonWithVersion(path)
	return lesson, err
}

func LoadLessonWithVersion(path string) (Lesson, cacheutil.DataVersion, error) {
	data, version, err := cacheutil.ReadFileWithVersion(path)
	if err != nil {
		return Lesson{}, cacheutil.DataVersion{}, err
	}

	var lesson Lesson
	if err := json.Unmarshal(data, &lesson); err != nil {
		return Lesson{}, cacheutil.DataVersion{}, err
	}
	if lesson.ID == "" {
		return Lesson{}, cacheutil.DataVersion{}, errors.New("lesson id is required")
	}

	return lesson, version, nil
}

func LoadRepository(path string) (*Repository, error) {
	data, version, err := cacheutil.ReadFileWithVersion(path)
	if err != nil {
		return nil, err
	}

	var loaded []Lesson
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, err
	}

	repo := &Repository{
		lessons:    loaded,
		byID:       make(map[string]Lesson, len(loaded)),
		summaries:  make([]Summary, 0, len(loaded)),
		version:    version,
		listJSON:   map[string][]byte{},
		lessonJSON: map[string][]byte{},
	}

	seenCategories := map[string]bool{}
	for _, lesson := range loaded {
		if lesson.ID == "" {
			return nil, errors.New("lesson id is required")
		}
		if _, exists := repo.byID[lesson.ID]; exists {
			return nil, errors.New("duplicate lesson id: " + lesson.ID)
		}
		repo.byID[lesson.ID] = lesson
		repo.summaries = append(repo.summaries, Summary{
			ID:         lesson.ID,
			Title:      lesson.Title,
			Category:   lesson.Category,
			Difficulty: lesson.Difficulty,
		})
		if !seenCategories[lesson.Category] {
			seenCategories[lesson.Category] = true
			repo.categories = append(repo.categories, lesson.Category)
		}
	}

	categoriesJSON, err := json.Marshal(repo.categories)
	if err != nil {
		return nil, err
	}
	repo.categoriesJSON = categoriesJSON

	return repo, nil
}

func (r *Repository) List(category string, query string, difficulty string) []Summary {
	category = strings.TrimSpace(strings.ToLower(category))
	query = strings.TrimSpace(strings.ToLower(query))
	difficulty = strings.TrimSpace(strings.ToLower(difficulty))

	summaries := make([]Summary, 0, len(r.summaries))
	for _, summary := range r.summaries {
		if category != "" && strings.ToLower(summary.Category) != category {
			continue
		}
		if difficulty != "" && strings.ToLower(summary.Difficulty) != difficulty {
			continue
		}
		if query != "" && !summaryMatchesQuery(summary, query) {
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries
}

func (r *Repository) Get(id string) (Lesson, bool) {
	lesson, ok := r.byID[id]
	return lesson, ok
}

func (r *Repository) Categories() []string {
	return append([]string(nil), r.categories...)
}

func (r *Repository) Version() cacheutil.DataVersion {
	return r.version
}

func (r *Repository) CategoriesBytes() []byte {
	return append([]byte(nil), r.categoriesJSON...)
}

func (r *Repository) ListBytes(category string, query string, difficulty string) ([]byte, error) {
	key := cacheutil.HashKey(category, query, difficulty)

	r.mu.RLock()
	if body, ok := r.listJSON[key]; ok {
		r.mu.RUnlock()
		return append([]byte(nil), body...), nil
	}
	r.mu.RUnlock()

	body, err := json.Marshal(r.List(category, query, difficulty))
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	if cached, ok := r.listJSON[key]; ok {
		r.mu.Unlock()
		return append([]byte(nil), cached...), nil
	}
	r.listJSON[key] = append([]byte(nil), body...)
	r.mu.Unlock()

	return body, nil
}

func (r *Repository) LessonBytes(id string) ([]byte, bool, error) {
	r.mu.RLock()
	if body, ok := r.lessonJSON[id]; ok {
		r.mu.RUnlock()
		return append([]byte(nil), body...), true, nil
	}
	r.mu.RUnlock()

	lesson, ok := r.Get(id)
	if !ok {
		return nil, false, nil
	}

	body, err := json.Marshal(lesson)
	if err != nil {
		return nil, false, err
	}

	r.mu.Lock()
	if cached, ok := r.lessonJSON[id]; ok {
		r.mu.Unlock()
		return append([]byte(nil), cached...), true, nil
	}
	r.lessonJSON[id] = append([]byte(nil), body...)
	r.mu.Unlock()

	return body, true, nil
}

func summaryMatchesQuery(summary Summary, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		summary.Title,
		summary.Category,
		summary.Difficulty,
	}, " "))

	return strings.Contains(haystack, query)
}
