package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hacknation-bot/internal/config"
	"hacknation-bot/internal/notifier"
	"hacknation-bot/internal/scraper"
	"hacknation-bot/internal/storage"
)

func main() {
	onceFlag := flag.Bool("once", false, "Wykonaj pojedyncze sprawdzenie i zakończ działanie")
	dryRunFlag := flag.Bool("dry-run", false, "Tryb testowy - loguj powiadomienia zamiast wysyłać na Discord")
	dailySummaryFlag := flag.Bool("daily-summary", false, "Wyślij natychmiast dzienne podsumowanie na Discord i zakończ")
	flag.Parse()

	log.Println("🚀 Uruchamianie HackNation Zadania Bot...")

	cfg := config.LoadConfig()

	if *dryRunFlag {
		log.Println("🧪 Uruchomiono w trybie DRY-RUN (powiadomienia będą wypisywane do konsoli)")
	}

	// Inicjalizacja bazy SQLite
	store, err := storage.NewStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("❌ Błąd inicjalizacji bazy danych SQLite (%s): %v", cfg.DBPath, err)
	}
	defer store.Close()
	log.Printf("💾 Baza danych SQLite zainicjalizowana pomyślnie: %s", cfg.DBPath)

	// Inicjalizacja scrapera
	scr := scraper.NewScraper(cfg.TargetURL, nil)

	// Inicjalizacja notyfikatora Discord
	notif := notifier.NewDiscordNotifier(cfg.DiscordWebhookURL, *dryRunFlag)
	if cfg.DiscordWebhookURL == "" && !*dryRunFlag {
		log.Println("⚠️ OSTRZEŻENIE: Brak DISCORD_WEBHOOK_URL w konfiguracji! Powiadomienia będą logowane lokalnie.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Tryb natychmiastowego wysłania dziennego podsumowania (--daily-summary)
	if *dailySummaryFlag {
		log.Println("📊 Wymuszone wysyłanie dziennego podsumowania (--daily-summary)...")
		if err := sendDailySummary(ctx, store, notif); err != nil {
			log.Fatalf("❌ Błąd podczas wysyłania dziennego podsumowania: %v", err)
		}
		return
	}

	// Pierwsze sprawdzenie natychmiast po uruchomieniu
	log.Println("🔍 Wykonywanie sprawdzania zadań...")
	runCheck(ctx, scr, store, notif, cfg)
	checkDailySummary(ctx, store, notif, cfg)

	if *onceFlag {
		log.Println("🏁 Zakończono pojedyncze sprawdzenie (--once). Wyłączanie.")
		return
	}

	// Pętla cyklicznego monitorowania
	log.Printf("⏰ Uruchomiono cykliczny monitoring (interwał: %v, dzienne podsumowanie: %v o %02d:%02d, tylko gdy brak zmian: %v). Oczekiwanie na cykl...",
		cfg.CheckInterval, cfg.DailySummaryEnabled, cfg.DailySummaryHour, cfg.DailySummaryMinute, cfg.DailySummaryOnlyIfNoChanges)
	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			log.Println("🔍 [Cykl] Rozpoczynanie cyklicznego sprawdzania zadań...")
			runCheck(ctx, scr, store, notif, cfg)
			checkDailySummary(ctx, store, notif, cfg)
		case sig := <-sigChan:
			log.Printf("🛑 Odebrano sygnał %v, bezpieczne wyłączanie bota...", sig)
			return
		}
	}
}

func runCheck(ctx context.Context, scr *scraper.Scraper, store *storage.Storage, notif *notifier.DiscordNotifier, cfg *config.Config) {
	tasks, err := scr.FetchTasks(ctx)
	if err != nil {
		log.Printf("❌ Błąd podczas pobierania zadań ze strony %s: %v", cfg.TargetURL, err)
		return
	}

	log.Printf("📥 Pobrano %d zadań ze strony %s", len(tasks), cfg.TargetURL)

	newTasks, updatedTasks, isInitialSync, err := store.SyncTasks(tasks)
	if err != nil {
		log.Printf("❌ Błąd podczas synchronizacji zadań w bazie: %v", err)
		return
	}

	if isInitialSync {
		log.Printf("🌱 Pierwsza synchronizacja: zapisano %d zadań jako stan początkowy", len(tasks))

		if cfg.NotifyOnInitialRun {
			log.Printf("📢 Wysyłanie powiadomień dla %d zadań początkowych (NOTIFY_ON_INITIAL_RUN=true)...", len(newTasks))
			var embeds []notifier.Embed
			for _, t := range newTasks {
				embeds = append(embeds, notifier.FormatNewTaskEmbed(t))
			}
			if err := notif.SendEmbeds(ctx, embeds); err != nil {
				log.Printf("❌ Błąd wysyłania powiadomień początkowych: %v", err)
			}
		} else if cfg.SendStartupMessage {
			log.Println("📢 Wysyłanie powiadomienia o starcie monitoringu...")
			startupEmbed := notifier.FormatStartupEmbed(len(tasks))
			if err := notif.SendEmbeds(ctx, []notifier.Embed{startupEmbed}); err != nil {
				log.Printf("❌ Błąd wysyłania powiadomienia o starcie: %v", err)
			}
		}
		return
	}

	// Kolejne synchronizacje
	if len(newTasks) == 0 && len(updatedTasks) == 0 {
		log.Println("✅ Brak zmian na stronie — wszystkie zadania są aktualne.")
		return
	}

	log.Printf("🔔 Wykryto zmiany: %d nowych zadań, %d zaktualizowanych", len(newTasks), len(updatedTasks))

	var embedsToSend []notifier.Embed

	for _, t := range newTasks {
		log.Printf("  ➕ Nowe zadanie: %s (ID: %s)", t.Title, t.ID)
		embedsToSend = append(embedsToSend, notifier.FormatNewTaskEmbed(t))
	}

	for _, diff := range updatedTasks {
		log.Printf("  ✏️ Aktualizacja zadania: %s (%d zmienionych pól)", diff.Task.Title, len(diff.FieldDiffs))
		for _, fd := range diff.FieldDiffs {
			log.Printf("     • %s: %s ➔ %s", fd.FieldName, fd.OldValue, fd.NewValue)
		}
		embedsToSend = append(embedsToSend, notifier.FormatUpdatedTaskEmbed(diff))
	}

	if len(embedsToSend) > 0 {
		if err := notif.SendEmbeds(ctx, embedsToSend); err != nil {
			log.Printf("❌ Błąd podczas wysyłania powiadomień na Discord: %v", err)
		} else {
			log.Printf("🚀 Pomyślnie wysłano %d powiadomień na Discord!", len(embedsToSend))
		}
	}
}

func checkDailySummary(ctx context.Context, store *storage.Storage, notif *notifier.DiscordNotifier, cfg *config.Config) {
	if !cfg.DailySummaryEnabled {
		return
	}

	now := time.Now()
	// Jeśli godzina:minuta jest wcześniejsza niż wyznaczony czas podsumowania, czekamy
	if now.Hour() < cfg.DailySummaryHour || (now.Hour() == cfg.DailySummaryHour && now.Minute() < cfg.DailySummaryMinute) {
		return
	}

	today := now.Format("2006-01-02")
	lastDate, err := store.GetLastDailySummaryDate()
	if err != nil {
		log.Printf("⚠️ Nie można odczytać daty ostatniego podsumowania dziennego: %v", err)
		return
	}

	if lastDate == today {
		// Dzisiejsze podsumowanie zostało już obsłużone
		return
	}

	newCount, updatedCount, err := store.GetActivityLast24Hours()
	if err != nil {
		log.Printf("⚠️ Błąd pobierania aktywności z bazy: %v", err)
		return
	}

	// Jeśli użytkownik zażądał wysyłania tylko w przypadku braku zmian w ciągu 24h:
	if cfg.DailySummaryOnlyIfNoChanges && (newCount > 0 || updatedCount > 0) {
		log.Printf("ℹ️ Pomijanie raportu dziennego (DAILY_SUMMARY_ONLY_IF_NO_CHANGES=true, wykryto %d zmian w 24h)", newCount+updatedCount)
		_ = store.SetLastDailySummaryDate(today)
		return
	}

	log.Printf("📊 Przygotowywanie codziennego raportu za dzień %s...", today)
	if err := sendDailySummary(ctx, store, notif); err != nil {
		log.Printf("❌ Błąd podczas wysyłania dziennego podsumowania: %v", err)
		return
	}

	if err := store.SetLastDailySummaryDate(today); err != nil {
		log.Printf("⚠️ Błąd podczas zapisywania daty podsumowania dziennego: %v", err)
	}
}

func sendDailySummary(ctx context.Context, store *storage.Storage, notif *notifier.DiscordNotifier) error {
	tasks, err := store.GetAllTasks()
	if err != nil {
		return fmt.Errorf("błąd pobierania zadań z bazy: %w", err)
	}

	newCount, updatedCount, err := store.GetActivityLast24Hours()
	if err != nil {
		return fmt.Errorf("błąd pobierania aktywności z bazy: %w", err)
	}

	embed := notifier.FormatDailySummaryEmbed(tasks, newCount, updatedCount)
	if err := notif.SendEmbeds(ctx, []notifier.Embed{embed}); err != nil {
		return fmt.Errorf("błąd wysyłania embeda na Discord: %w", err)
	}

	log.Printf("📊 Wysłano dzienne podsumowanie na Discord (zadania: %d, aktywność 24h: nowe=%d, zaktualizowane=%d)",
		len(tasks), newCount, updatedCount)
	return nil
}
