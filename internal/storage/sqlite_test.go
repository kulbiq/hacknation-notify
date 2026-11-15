package storage

import (
	"testing"

	"hacknation-bot/internal/scraper"
)

func TestStorageSync(t *testing.T) {
	s, err := NewStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer s.Close()

	task1 := scraper.Task{
		ID:          "risk_miner",
		Title:       "RISK MINER",
		FileName:    "risk_miner.md",
		Initiator:   "MINISTERSTWO FUNDUSZY",
		Campaign:    "RISK MINER",
		Description: "Opis 1",
		Prize:       "Brak informacji",
		LogoURL:     "https://example.com/logo1.png",
	}
	task1.ContentHash = task1.ComputeHash()

	task2 := scraper.Task{
		ID:          "radar_innowacji",
		Title:       "RADAR INNOWACJI",
		FileName:    "radar_innowacji.md",
		Initiator:   "PODKARPACKIE",
		Campaign:    "RADAR INNOWACJI",
		Description: "Opis 2",
		Prize:       "11 111 PLN",
		LogoURL:     "https://example.com/logo2.png",
	}
	task2.ContentHash = task2.ComputeHash()

	// 1. First sync (empty database)
	newTasks, updatedTasks, isInitialSync, err := s.SyncTasks([]scraper.Task{task1, task2})
	if err != nil {
		t.Fatalf("SyncTasks failed: %v", err)
	}
	if !isInitialSync {
		t.Errorf("expected isInitialSync to be true")
	}
	if len(newTasks) != 2 {
		t.Errorf("expected 2 new tasks, got %d", len(newTasks))
	}
	if len(updatedTasks) != 0 {
		t.Errorf("expected 0 updated tasks, got %d", len(updatedTasks))
	}

	count, err := s.GetTaskCount()
	if err != nil || count != 2 {
		t.Errorf("expected 2 tasks in DB, got count=%d, err=%v", count, err)
	}

	// 2. Second sync with exact same data
	newTasks2, updatedTasks2, isInitialSync2, err := s.SyncTasks([]scraper.Task{task1, task2})
	if err != nil {
		t.Fatalf("second SyncTasks failed: %v", err)
	}
	if isInitialSync2 {
		t.Errorf("expected isInitialSync to be false")
	}
	if len(newTasks2) != 0 {
		t.Errorf("expected 0 new tasks, got %d", len(newTasks2))
	}
	if len(updatedTasks2) != 0 {
		t.Errorf("expected 0 updated tasks, got %d", len(updatedTasks2))
	}

	// 3. Update task1: prize revealed!
	task1Updated := task1
	task1Updated.Prize = "25 000 PLN"
	task1Updated.ContentHash = task1Updated.ComputeHash()

	newTasks3, updatedTasks3, isInitialSync3, err := s.SyncTasks([]scraper.Task{task1Updated, task2})
	if err != nil {
		t.Fatalf("third SyncTasks failed: %v", err)
	}
	if isInitialSync3 {
		t.Errorf("expected isInitialSync to be false")
	}
	if len(newTasks3) != 0 {
		t.Errorf("expected 0 new tasks, got %d", len(newTasks3))
	}
	if len(updatedTasks3) != 1 {
		t.Fatalf("expected 1 updated task, got %d", len(updatedTasks3))
	}

	diff := updatedTasks3[0]
	if diff.Task.ID != "risk_miner" {
		t.Errorf("expected updated task ID 'risk_miner', got '%s'", diff.Task.ID)
	}
	if len(diff.FieldDiffs) != 1 {
		t.Fatalf("expected 1 field diff, got %d", len(diff.FieldDiffs))
	}
	fd := diff.FieldDiffs[0]
	if fd.FieldName != "Pula nagród" || fd.OldValue != "Brak informacji" || fd.NewValue != "25 000 PLN" {
		t.Errorf("unexpected field diff: %+v", fd)
	}
}
