package notifier

import (
	"strings"
	"testing"

	"hacknation-bot/internal/scraper"
)

func TestFormatNewTaskEmbed(t *testing.T) {
	task := scraper.Task{
		ID:          "risk_miner",
		Title:       "RISK MINER",
		FileName:    "risk_miner.md",
		Initiator:   "MINISTERSTWO FUNDUSZY",
		Campaign:    "RISK MINER",
		Description: "Przykładowy opis",
		Prize:       "25 000 PLN",
		LogoURL:     "https://example.com/logo.png",
	}

	embed := FormatNewTaskEmbed(task)

	if !strings.Contains(embed.Title, "RISK MINER") {
		t.Errorf("expected Title to contain 'RISK MINER', got '%s'", embed.Title)
	}
	if embed.Color != ColorGreen {
		t.Errorf("expected ColorGreen, got %x", embed.Color)
	}
	if embed.Thumbnail == nil || embed.Thumbnail.URL != task.LogoURL {
		t.Errorf("expected thumbnail to be set with logo URL")
	}

	foundPrize := false
	for _, f := range embed.Fields {
		if strings.Contains(f.Name, "Pula nagród") && f.Value == "25 000 PLN" {
			foundPrize = true
			break
		}
	}
	if !foundPrize {
		t.Errorf("expected to find prize field with '25 000 PLN'")
	}
}

func TestFormatUpdatedTaskEmbed(t *testing.T) {
	diff := scraper.TaskDiff{
		Task: scraper.Task{
			ID:    "risk_miner",
			Title: "RISK MINER",
			Prize: "25 000 PLN",
		},
		FieldDiffs: []scraper.FieldDiff{
			{
				FieldName: "Pula nagród",
				OldValue:  "Brak informacji",
				NewValue:  "25 000 PLN",
			},
		},
	}

	embed := FormatUpdatedTaskEmbed(diff)

	if !strings.Contains(embed.Title, "RISK MINER") {
		t.Errorf("expected Title to contain 'RISK MINER', got '%s'", embed.Title)
	}
	if embed.Color != ColorOrange {
		t.Errorf("expected ColorOrange, got %x", embed.Color)
	}

	foundChange := false
	for _, f := range embed.Fields {
		if strings.Contains(f.Name, "Wykryte zmiany") && strings.Contains(f.Value, "Brak informacji") && strings.Contains(f.Value, "25 000 PLN") {
			foundChange = true
			break
		}
	}
	if !foundChange {
		t.Errorf("expected to find change diff in fields")
	}
}
