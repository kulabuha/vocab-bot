package stats

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStore_RecordAdd_Get(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	// Initially zero
	u := s.Get(123)
	if u.AddRequests != 0 || u.WordsAdded != 0 || u.CollocationsAdded != 0 {
		t.Errorf("initial Get: got %+v", u)
	}
	// Record add
	s.RecordAdd(123, 2, 10)
	u = s.Get(123)
	if u.AddRequests != 1 || u.WordsAdded != 2 || u.CollocationsAdded != 10 {
		t.Errorf("after RecordAdd(123,2,10): got %+v", u)
	}
	if u.LastRequestAt.IsZero() {
		t.Error("LastRequestAt should be set")
	}
	// Second add
	s.RecordAdd(123, 1, 5)
	u = s.Get(123)
	if u.AddRequests != 2 || u.WordsAdded != 3 || u.CollocationsAdded != 15 {
		t.Errorf("after second add: got %+v", u)
	}
	// Different user
	s.RecordAdd(456, 1, 3)
	u2 := s.Get(456)
	if u2.AddRequests != 1 || u2.CollocationsAdded != 3 {
		t.Errorf("user 456: got %+v", u2)
	}
	// First user unchanged
	u = s.Get(123)
	if u.AddRequests != 2 {
		t.Errorf("user 123 unchanged: got %+v", u)
	}
}

func TestFileStore_RecordTrain_RecordAnswer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s.RecordTrain(100)
	s.RecordTrain(100)
	s.RecordAnswer(100)
	u := s.Get(100)
	if u.TrainRequests != 2 || u.ExercisesAnswered != 1 {
		t.Errorf("got %+v", u)
	}
}

func TestFileStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	s1, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.RecordAdd(999, 3, 15)
	s1.RecordTrain(999)
	// New store from same file
	s2, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	u := s2.Get(999)
	if u.AddRequests != 1 || u.WordsAdded != 3 || u.CollocationsAdded != 15 || u.TrainRequests != 1 {
		t.Errorf("after reload: got %+v", u)
	}
}

func TestFileStore_NewFileStore_missingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Get(1).AddRequests != 0 {
		t.Error("new store should have zero stats")
	}
}

func TestUserStats_ZeroValue(t *testing.T) {
	var u UserStats
	if u.AddRequests != 0 || !u.LastRequestAt.IsZero() {
		t.Errorf("zero value: %+v", u)
	}
}

func TestNopRecorder(t *testing.T) {
	var r NopRecorder
	r.RecordAdd(1, 2, 3)
	r.RecordTrain(1)
	r.RecordAnswer(1)
	// No panic
}

func TestUserStats_JSONRoundTrip(t *testing.T) {
	u := UserStats{
		AddRequests:       2,
		WordsAdded:        5,
		CollocationsAdded: 20,
		TrainRequests:     10,
		ExercisesAnswered: 8,
		LastRequestAt:     time.Date(2025, 3, 9, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "2025-03-09") {
		t.Errorf("expected RFC3339 date in JSON: %s", raw)
	}
	var u2 UserStats
	if err := json.Unmarshal(raw, &u2); err != nil {
		t.Fatal(err)
	}
	if u2.AddRequests != u.AddRequests || u2.CollocationsAdded != u.CollocationsAdded {
		t.Errorf("round-trip: got %+v", u2)
	}
}
