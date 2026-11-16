package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hacknation-bot/internal/scraper"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(dbPath string) (*Storage, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// Jeśli próba utworzenia /data na hoście zawiedzie, użyj katalogu lokalnego ./data
			fallbackDir := "./data"
			if mkErr := os.MkdirAll(fallbackDir, 0755); mkErr == nil {
				dbPath = filepath.Join(fallbackDir, filepath.Base(dbPath))
			} else {
				return nil, fmt.Errorf("failed to create directory for database: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Single connection pool recommended for SQLite write operations
	db.SetMaxOpenConns(1)

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return s, nil
}

func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		file_name TEXT,
		initiator TEXT,
		campaign TEXT,
		description TEXT,
		prize TEXT,
		logo_url TEXT,
		content_hash TEXT NOT NULL,
		first_seen_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sync_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		synced_at DATETIME NOT NULL,
		tasks_count INTEGER NOT NULL,
		new_count INTEGER NOT NULL,
		updated_count INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS bot_metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) GetLastDailySummaryDate() (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM bot_metadata WHERE key = 'last_daily_summary_date'").Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (s *Storage) SetLastDailySummaryDate(dateStr string) error {
	_, err := s.db.Exec(`
		INSERT INTO bot_metadata (key, value, updated_at)
		VALUES ('last_daily_summary_date', ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, dateStr, time.Now().UTC())
	return err
}

func (s *Storage) GetAllTasks() ([]scraper.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, title, file_name, initiator, campaign, description, prize, logo_url, content_hash
		FROM tasks
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []scraper.Task
	for rows.Next() {
		var t scraper.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.FileName, &t.Initiator, &t.Campaign, &t.Description, &t.Prize, &t.LogoURL, &t.ContentHash); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *Storage) GetActivityLast24Hours() (int, int, error) {
	var newCount, updatedCount int
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(new_count), 0), COALESCE(SUM(updated_count), 0)
		FROM sync_history
		WHERE synced_at >= datetime('now', '-24 hours')
	`).Scan(&newCount, &updatedCount)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return newCount, updatedCount, err
}

func (s *Storage) GetTaskCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
	return count, err
}

// SyncTasks updates tasks in DB and returns new and updated tasks.
// isInitialSync is true when the database was completely empty before this sync.
func (s *Storage) SyncTasks(scrapedTasks []scraper.Task) (newTasks []scraper.Task, updatedTasks []scraper.TaskDiff, isInitialSync bool, err error) {
	existingTasks, err := s.getAllTasksMap()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get existing tasks: %w", err)
	}

	isInitialSync = len(existingTasks) == 0
	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	insertStmt, err := tx.Prepare(`
		INSERT INTO tasks (id, title, file_name, initiator, campaign, description, prize, logo_url, content_hash, first_seen_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	updateStmt, err := tx.Prepare(`
		UPDATE tasks
		SET title = ?, file_name = ?, initiator = ?, campaign = ?, description = ?, prize = ?, logo_url = ?, content_hash = ?, updated_at = ?
		WHERE id = ?
	`)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	for _, task := range scrapedTasks {
		existing, found := existingTasks[task.ID]
		if !found {
			// New Task
			_, err := insertStmt.Exec(
				task.ID,
				task.Title,
				task.FileName,
				task.Initiator,
				task.Campaign,
				task.Description,
				task.Prize,
				task.LogoURL,
				task.ContentHash,
				now,
				now,
			)
			if err != nil {
				return nil, nil, false, fmt.Errorf("failed to insert task %s: %w", task.ID, err)
			}
			newTasks = append(newTasks, task)
		} else if existing.ContentHash != task.ContentHash {
			// Task changed
			diff := compareTasks(existing, task)
			if diff.HasChanges() {
				_, err := updateStmt.Exec(
					task.Title,
					task.FileName,
					task.Initiator,
					task.Campaign,
					task.Description,
					task.Prize,
					task.LogoURL,
					task.ContentHash,
					now,
					task.ID,
				)
				if err != nil {
					return nil, nil, false, fmt.Errorf("failed to update task %s: %w", task.ID, err)
				}
				updatedTasks = append(updatedTasks, diff)
			}
		}
	}

	// Record sync history
	_, err = tx.Exec(`
		INSERT INTO sync_history (synced_at, tasks_count, new_count, updated_count)
		VALUES (?, ?, ?, ?)
	`, now, len(scrapedTasks), len(newTasks), len(updatedTasks))
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to record sync history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newTasks, updatedTasks, isInitialSync, nil
}

func (s *Storage) getAllTasksMap() (map[string]scraper.Task, error) {
	rows, err := s.db.Query(`
		SELECT id, title, file_name, initiator, campaign, description, prize, logo_url, content_hash
		FROM tasks
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]scraper.Task)
	for rows.Next() {
		var t scraper.Task
		err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.FileName,
			&t.Initiator,
			&t.Campaign,
			&t.Description,
			&t.Prize,
			&t.LogoURL,
			&t.ContentHash,
		)
		if err != nil {
			return nil, err
		}
		result[t.ID] = t
	}
	return result, rows.Err()
}

func compareTasks(oldTask, newTask scraper.Task) scraper.TaskDiff {
	diff := scraper.TaskDiff{
		Task:    newTask,
		OldTask: oldTask,
	}

	if oldTask.Title != newTask.Title {
		diff.FieldDiffs = append(diff.FieldDiffs, scraper.FieldDiff{
			FieldName: "Tytuł",
			OldValue:  oldTask.Title,
			NewValue:  newTask.Title,
		})
	}
	if oldTask.Prize != newTask.Prize {
		diff.FieldDiffs = append(diff.FieldDiffs, scraper.FieldDiff{
			FieldName: "Pula nagród",
			OldValue:  oldTask.Prize,
			NewValue:  newTask.Prize,
		})
	}
	if oldTask.Initiator != newTask.Initiator {
		diff.FieldDiffs = append(diff.FieldDiffs, scraper.FieldDiff{
			FieldName: "Inicjator",
			OldValue:  oldTask.Initiator,
			NewValue:  newTask.Initiator,
		})
	}
	if oldTask.Campaign != newTask.Campaign {
		diff.FieldDiffs = append(diff.FieldDiffs, scraper.FieldDiff{
			FieldName: "Kampania",
			OldValue:  oldTask.Campaign,
			NewValue:  newTask.Campaign,
		})
	}
	if oldTask.Description != newTask.Description {
		diff.FieldDiffs = append(diff.FieldDiffs, scraper.FieldDiff{
			FieldName: "Opis",
			OldValue:  oldTask.Description,
			NewValue:  newTask.Description,
		})
	}
	if oldTask.LogoURL != newTask.LogoURL {
		diff.FieldDiffs = append(diff.FieldDiffs, scraper.FieldDiff{
			FieldName: "Logo",
			OldValue:  oldTask.LogoURL,
			NewValue:  newTask.LogoURL,
		})
	}

	return diff
}
