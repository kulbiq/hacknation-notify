# --- Etap budowania ---
FROM golang:alpine AS builder

WORKDIR /app

# Instalacja certyfikatów CA
RUN apk add --no-cache ca-certificates git

# Kopiowanie plików definicji modułów i pobranie zależności
COPY go.mod go.sum ./
RUN go mod download

# Kopiowanie kodu źródłowego
COPY . .

# Kompilacja statycznego binarza bez CGO
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /hacknation-bot ./cmd/bot

# --- Etap uruchomieniowy ---
FROM alpine:3.21

# Dodanie certyfikatów do komunikacji HTTPS oraz stref czasowych
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Katalog na dane SQLite
RUN mkdir -p /data

# Kopiowanie binarza z etapu budowania
COPY --from=builder /hacknation-bot /app/hacknation-bot

# Domyślny wolumen na bazę danych
VOLUME ["/data"]

ENV DB_PATH=/data/hacknation.db

ENTRYPOINT ["/app/hacknation-bot"]
