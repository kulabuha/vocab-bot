// Package stats provides per-user usage statistics persisted to a JSON file.
package stats

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"time"
)

// UserStats holds per-user counters and last activity.
type UserStats struct {
	AddRequests       int64     `json:"add_requests"`        // number of /add (or add_words) requests
	WordsAdded        int64     `json:"words_added"`         // total valid words submitted in add
	CollocationsAdded int64     `json:"collocations_added"`  // total new collocations created/reused
	TrainRequests     int64     `json:"train_requests"`      // number of /train (next exercise) requests
	ExercisesAnswered int64     `json:"exercises_answered"`  // number of graded answers (correct or wrong)
	LastRequestAt     time.Time `json:"last_request_at"`    // time of last recorded action
}

// FileStore persists per-user stats to a JSON file. Safe for concurrent use.
// Keys are chat_id as string for JSON compatibility.
type FileStore struct {
	mu   sync.Mutex
	path string
	data map[string]UserStats
}

// NewFileStore loads existing data from path or starts empty. Path is written on each Record.
func NewFileStore(path string) (*FileStore, error) {
	s := &FileStore{path: path, data: make(map[string]UserStats)}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &s.data)
}

func (s *FileStore) key(chatID int64) string { return strconv.FormatInt(chatID, 10) }

func (s *FileStore) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// RecordAdd records an add-words request: increments add_requests, adds to words_added and collocations_added, updates last_request_at.
func (s *FileStore) RecordAdd(chatID int64, wordsAdded, collocationsAdded int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(chatID)
	u := s.data[k]
	u.AddRequests++
	u.WordsAdded += int64(wordsAdded)
	u.CollocationsAdded += int64(collocationsAdded)
	u.LastRequestAt = time.Now().UTC()
	s.data[k] = u
	_ = s.save()
}

// RecordTrain records a train (next exercise) request.
func (s *FileStore) RecordTrain(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(chatID)
	u := s.data[k]
	u.TrainRequests++
	u.LastRequestAt = time.Now().UTC()
	s.data[k] = u
	_ = s.save()
}

// RecordAnswer records a graded answer (exercise completed).
func (s *FileStore) RecordAnswer(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(chatID)
	u := s.data[k]
	u.ExercisesAnswered++
	u.LastRequestAt = time.Now().UTC()
	s.data[k] = u
	_ = s.save()
}

// Get returns a copy of stats for the user, or zero stats if never seen.
func (s *FileStore) Get(chatID int64) UserStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[s.key(chatID)]
}

// Recorder is the interface used by handlers to record events. No-op if nil.
type Recorder interface {
	RecordAdd(chatID int64, wordsAdded, collocationsAdded int)
	RecordTrain(chatID int64)
	RecordAnswer(chatID int64)
}

// NopRecorder does nothing (for when stats file is not configured).
type NopRecorder struct{}

func (NopRecorder) RecordAdd(chatID int64, wordsAdded, collocationsAdded int) {}
func (NopRecorder) RecordTrain(chatID int64)                                 {}
func (NopRecorder) RecordAnswer(chatID int64)                               {}
