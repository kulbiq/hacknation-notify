package scraper

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Task represents a hackathon challenge from hacknation.pl/zadania
type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	FileName    string `json:"file_name"`
	Initiator   string `json:"initiator"`
	Campaign    string `json:"campaign"`
	Description string `json:"description"`
	Prize       string `json:"prize"`
	LogoURL     string `json:"logo_url"`
	ContentHash string `json:"content_hash"`
}

// ComputeHash computes a SHA-256 hash of the task's functional fields.
func (t *Task) ComputeHash() string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		strings.TrimSpace(t.Title),
		strings.TrimSpace(t.FileName),
		strings.TrimSpace(t.Initiator),
		strings.TrimSpace(t.Campaign),
		strings.TrimSpace(t.Description),
		strings.TrimSpace(t.Prize),
		strings.TrimSpace(t.LogoURL),
	)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// TaskDiff represents detected changes between an existing task and a newly scraped task.
type TaskDiff struct {
	Task        Task
	OldTask     Task
	FieldDiffs  []FieldDiff
}

// FieldDiff represents a single field change
type FieldDiff struct {
	FieldName string
	OldValue  string
	NewValue  string
}

// HasChanges returns true if there are any field diffs
func (td *TaskDiff) HasChanges() bool {
	return len(td.FieldDiffs) > 0
}
