package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	ColorGreen  = 0x2ecc71
	ColorOrange = 0xf39c12
	ColorBlue   = 0x3498db
	ColorRed    = 0xe74c3c

	BotUsername  = "HackNation Bot"
	BotAvatarURL = "https://hacknation.pl/favicon.png"
)

type WebhookPayload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	URL         string        `json:"url,omitempty"`
	Color       int           `json:"color,omitempty"`
	Fields      []EmbedField  `json:"fields,omitempty"`
	Thumbnail   *EmbedMedia   `json:"thumbnail,omitempty"`
	Footer      *EmbedFooter  `json:"footer,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedMedia struct {
	URL string `json:"url"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type DiscordNotifier struct {
	webhookURL string
	httpClient *http.Client
	dryRun     bool
}

func NewDiscordNotifier(webhookURL string, dryRun bool) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		dryRun:     dryRun || webhookURL == "",
	}
}

// SendEmbeds sends one or more embeds to Discord. Discord allows max 10 embeds per message.
func (n *DiscordNotifier) SendEmbeds(ctx context.Context, embeds []Embed) error {
	if len(embeds) == 0 {
		return nil
	}

	// Discord limit: max 10 embeds per webhook call
	const chunkSize = 10
	for i := 0; i < len(embeds); i += chunkSize {
		end := i + chunkSize
		if end > len(embeds) {
			end = len(embeds)
		}
		chunk := embeds[i:end]

		payload := WebhookPayload{
			Username:  BotUsername,
			AvatarURL: BotAvatarURL,
			Embeds:    chunk,
		}

		if err := n.sendPayload(ctx, payload); err != nil {
			return err
		}

		// Brief delay to prevent hitting rate limits
		if end < len(embeds) {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func (n *DiscordNotifier) SendMessage(ctx context.Context, content string) error {
	payload := WebhookPayload{
		Username:  BotUsername,
		AvatarURL: BotAvatarURL,
		Content:   content,
	}
	return n.sendPayload(ctx, payload)
}

func (n *DiscordNotifier) sendPayload(ctx context.Context, payload WebhookPayload) error {
	if n.dryRun {
		jsonData, _ := json.MarshalIndent(payload, "", "  ")
		log.Printf("[DRY-RUN / NO WEBHOOK] Discord payload:\n%s\n", string(jsonData))
		return nil
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
