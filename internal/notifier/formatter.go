package notifier

import (
	"fmt"
	"strings"
	"time"

	"hacknation-bot/internal/scraper"
)

const TargetTasksURL = "https://hacknation.pl/zadania"

// FormatNewTaskEmbed formats a newly discovered task into a rich Discord embed in Polish.
func FormatNewTaskEmbed(task scraper.Task) Embed {
	var fields []EmbedField

	if task.Initiator != "" {
		fields = append(fields, EmbedField{
			Name:   "🏢 Inicjator projektu",
			Value:  task.Initiator,
			Inline: true,
		})
	}

	prizeVal := task.Prize
	if prizeVal == "" {
		prizeVal = "Brak informacji"
	}
	fields = append(fields, EmbedField{
		Name:   "🏆 Pula nagród",
		Value:  prizeVal,
		Inline: true,
	})

	if task.FileName != "" {
		fields = append(fields, EmbedField{
			Name:   "📁 Plik zadania",
			Value:  fmt.Sprintf("`%s`", task.FileName),
			Inline: true,
		})
	}

	if task.Description != "" {
		desc := task.Description
		if len(desc) > 950 {
			desc = desc[:950] + "..."
		}
		fields = append(fields, EmbedField{
			Name:   "📝 Opis wyzwania",
			Value:  desc,
			Inline: false,
		})
	}

	var thumbnail *EmbedMedia
	if task.LogoURL != "" {
		thumbnail = &EmbedMedia{URL: task.LogoURL}
	}

	return Embed{
		Title:       fmt.Sprintf("🆕 Nowe zadanie: %s", task.Title),
		Description: fmt.Sprintf("Na stronie [hacknation.pl/zadania](%s) pojawiło się nowe zadanie!", TargetTasksURL),
		URL:         TargetTasksURL,
		Color:       ColorGreen,
		Fields:      fields,
		Thumbnail:   thumbnail,
		Footer: &EmbedFooter{
			Text: "HackNation 2026 • Wyzwania partnerskie",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// FormatUpdatedTaskEmbed formats a modified task into a rich Discord embed highlighting what changed.
func FormatUpdatedTaskEmbed(diff scraper.TaskDiff) Embed {
	var fields []EmbedField

	if diff.Task.Initiator != "" {
		fields = append(fields, EmbedField{
			Name:   "🏢 Inicjator",
			Value:  diff.Task.Initiator,
			Inline: true,
		})
	}

	fields = append(fields, EmbedField{
		Name:   "🏆 Aktualna pula nagród",
		Value:  diff.Task.Prize,
		Inline: true,
	})

	var changeLines []string
	for _, fd := range diff.FieldDiffs {
		oldVal := fd.OldValue
		newVal := fd.NewValue
		if len(oldVal) > 60 {
			oldVal = oldVal[:57] + "..."
		}
		if len(newVal) > 60 {
			newVal = newVal[:57] + "..."
		}
		changeLines = append(changeLines, fmt.Sprintf("• **%s**: `%s` ➔ `%s`", fd.FieldName, oldVal, newVal))
	}

	fields = append(fields, EmbedField{
		Name:   "🔄 Wykryte zmiany",
		Value:  strings.Join(changeLines, "\n"),
		Inline: false,
	})

	var thumbnail *EmbedMedia
	if diff.Task.LogoURL != "" {
		thumbnail = &EmbedMedia{URL: diff.Task.LogoURL}
	}

	return Embed{
		Title:       fmt.Sprintf("✏️ Aktualizacja zadania: %s", diff.Task.Title),
		Description: fmt.Sprintf("Zaktualizowano dane zadania na stronie [hacknation.pl/zadania](%s).", TargetTasksURL),
		URL:         TargetTasksURL,
		Color:       ColorOrange,
		Fields:      fields,
		Thumbnail:   thumbnail,
		Footer: &EmbedFooter{
			Text: "HackNation 2026 • Zmiana w zadaniu",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// FormatStartupEmbed creates a notification when the bot starts and connects.
func FormatStartupEmbed(taskCount int) Embed {
	return Embed{
		Title: "🤖 HackNation Bot — Monitoring aktywny",
		Description: fmt.Sprintf(
			"Bot pomyślnie wystartował i monitoruje stronę [hacknation.pl/zadania](%s).\n\nAktualny stan bazy: **%d zadań**.",
			TargetTasksURL,
			taskCount,
		),
		URL:   TargetTasksURL,
		Color: ColorBlue,
		Footer: &EmbedFooter{
			Text: "HackNation 2026 Bot",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
