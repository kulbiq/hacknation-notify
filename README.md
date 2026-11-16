# 🤖 HackNation Zadania — Discord Bot (Go)

Bot w języku **Go** monitorujący stronę z zadaniami hackathonu [hacknation.pl/zadania](https://hacknation.pl/zadania).  
Pobiera i parsuje karty wyzwań (`<article class="hn-tterm-card ...">`), zapisuje stan w bazie **SQLite** i wysyła estetyczne powiadomienia (Rich Embeds) w języku polskim na kanał **Discord** za pomocą Webhooka, informując o nowych zadaniach i aktualizacjach (np. ogłoszeniu puli nagród, zmianie opisu czy inicjatora).

---

## ✨ Funkcjonalności

- **Precyzyjne parsowanie HTML**: Ekstrakcja tytułu, inicjatora, kampanii, pliku zadania, opisu, puli nagród oraz miniatury logo bezpośrednio z elementów DOM strony.
- **Inteligentne wykrywanie zmian (Diff)**: Hash SHA-256 oraz porównywanie poszczególnych pól. Bot informuje dokładnie o tym, co uległo zmianie (np. `Pula nagród: Brak informacji ➔ 25 000 PLN`).
- **Historia w SQLite (Pure Go)**: Użycie sterownika `modernc.org/sqlite` niewymagającego CGO – prosta i bezproblemowa kompilacja w dowolnym środowisku oraz w Dockerze.
- **Powiadomienia na Discord (Rich Embeds)**:
  - 🆕 **Nowe zadanie** (zielony embed z logo, inicjatorem, nagrodą i opisem).
  - ✏️ **Aktualizacja zadania** (pomarańczowy embed z wyszczególnionymi zmianami).
  - 📊 **Codzienny raport (Daily Summary)** (raport o stałej godzinie informujący np. o braku zmian w ciągu 24h lub podsumowujący aktywność).
  - 🤖 **Status startowy** (informacja o aktywności monitoringu i liczbie załadowanych zadań).
- **Proste wdrożenie**: Gotowy `Dockerfile` (multi-stage) i `docker-compose.yml`.
- **Tryby działania**: Cykliczny daemon lub jednorazowe sprawdzenie (`--once`), wymuszenie raportu dziennego (`--daily-summary`), a także tryb testowy (`--dry-run`).

---

## 🚀 Szybki start (Docker Compose)

### 1. Klonowanie i konfiguracja

Skopiuj przykładowy plik konfiguracyjny `.env.example` do `.env`:
```bash
cp .env.example .env
```

Wyedytuj plik `.env` i wklej swój adres Webhooka z Discorda:
```dotenv
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/TWOJ_ID/TWOJ_TOKEN
CHECK_INTERVAL=15m
NOTIFY_ON_INITIAL_RUN=false
SEND_STARTUP_MESSAGE=true
```

> 💡 **Jak stworzyć Webhook w Discordzie?**  
> 1. Wejdź w ustawienia wybranego kanału tekstowego na Discordzie.  
> 2. Kliknij **Integracje** -> **Webhooki** -> **Nowy webhook**.  
> 3. Skopiuj adres URL webhooka i wklej do zmiennej `DISCORD_WEBHOOK_URL`.

### 2. Uruchomienie bota

```bash
docker compose up -d
```

### 3. Sprawdzenie logów

```bash
docker compose logs -f
```

---

## 🛠️ Uruchomienie lokalne (bez Dockera)

Wymagane: Go 1.22+.

1. Pobierz zależności:
   ```bash
   go mod download
   ```

2. Skopiuj i ustaw plik `.env`:
   ```bash
   cp .env.example .env
   ```

3. Uruchom bota:
   ```bash
   go run ./cmd/bot
   ```

### Przydatne flagi:
- **Pojedyncze sprawdzenie i wyjście**:
  ```bash
  go run ./cmd/bot --once
  ```
- **Wymuszenie wysłania raportu dziennego i wyjście**:
  ```bash
  go run ./cmd/bot --daily-summary
  ```
- **Tryb testowy (DRY-RUN)** – nie wysyła zapytań do Discorda, tylko loguje treść embeda do konsoli:
  ```bash
  go run ./cmd/bot --once --dry-run
  ```

---

## ⚙️ Zmienne środowiskowe

| Zmienna | Domyślna wartość | Opis |
|---|---|---|
| `DISCORD_WEBHOOK_URL` | *(brak)* | URL webhooka Discorda do wysyłania powiadomień |
| `CHECK_INTERVAL` | `15m` | Odstęp czasu między kolejnymi sprawdzeniami (np. `5m`, `15m`, `1h`) |
| `DAILY_SUMMARY_ENABLED` | `true` | Czy wysyłać codzienne podsumowanie stanu bazy |
| `DAILY_SUMMARY_TIME` | `10:00` | Godzina wysyłki raportu dziennego (np. `10:00`, `18:30` lub sama godzina `0-23`) |
| `DAILY_SUMMARY_ONLY_IF_NO_CHANGES` | `false` | Jeśli `true`, raport zostanie wysłany **tylko** wtedy, gdy w ostatnich 24h nic nie dodano ani nie zmieniono |
| `TZ` | `Europe/Warsaw` | Strefa czasowa dla kontenera (zalecana strefa polska) |
| `TARGET_URL` | `https://hacknation.pl/zadania` | Adres monitorowanej strony z zadaniami |
| `DB_PATH` | `./data/hacknation.db` | Ścieżka do pliku bazy danych SQLite |
| `NOTIFY_ON_INITIAL_RUN` | `false` | Czy przy pierwszym uruchomieniu wysłać powiadomienia dla wszystkich istniejących zadań |
| `SEND_STARTUP_MESSAGE` | `true` | Czy wysłać powiadomienie powitalne po pomyślnym pierwszym sprawdzeniu |

---

## 📁 Struktura katalogów

```
hacknation-bot/
├── cmd/
│   └── bot/
│       └── main.go               # Punkt wejścia bota, pętla monitorowania, sygnały OS
├── internal/
│   ├── config/
│   │   └── config.go             # Ładowanie konfiguracji ze środowiska / .env
│   ├── scraper/
│   │   ├── model.go              # Model danych Task, TaskDiff, ContentHash
│   │   ├── scraper.go            # Pobieranie i parsowanie HTML (goquery)
│   │   └── scraper_test.go       # Testy jednostkowe parsowania kart zadań
│   ├── storage/
│   │   ├── sqlite.go             # Zarządzanie bazą SQLite, wykrywanie zmian
│   │   └── sqlite_test.go        # Testy synchronizacji i diffów
│   └── notifier/
│       ├── discord.go            # Klient wysyłki Webhooków Discord
│       ├── formatter.go          # Formatowanie Rich Embeds po polsku
│       └── formatter_test.go     # Testy formatowania komunikatów
├── data/                         # Wolumen na bazę SQLite (montowany w Dockerze)
├── .env.example                  # Szablon pliku konfiguracyjnego
├── Dockerfile                    # Wielopoziomowy (multi-stage) lekki obraz Docker
├── docker-compose.yml            # Konfiguracja Docker Compose
└── README.md
```

---

## 🧪 Testy

Uruchomienie wszystkich testów jednostkowych:
```bash
go test -v ./...
```
