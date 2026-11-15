package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	DefaultTargetURL = "https://hacknation.pl/zadania"
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 HackNationBot/1.0"
)

var (
	nonAlphaNumericRegex = regexp.MustCompile(`[^a-z0-9_-]+`)
	multipleSpacesRegex  = regexp.MustCompile(`\s+`)
)

type Scraper struct {
	targetURL  string
	userAgent  string
	httpClient *http.Client
}

func NewScraper(targetURL string, client *http.Client) *Scraper {
	if targetURL == "" {
		targetURL = DefaultTargetURL
	}
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Scraper{
		targetURL:  targetURL,
		userAgent:  DefaultUserAgent,
		httpClient: client,
	}
}

// FetchTasks connects to hacknation.pl/zadania and parses all tasks.
func (s *Scraper) FetchTasks(ctx context.Context) ([]Task, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pl,en-US;q=0.9,en;q=0.8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	return s.ParseHTML(resp.Body)
}

// ParseHTML parses an HTML reader containing hacknation zadania.
func (s *Scraper) ParseHTML(r io.Reader) ([]Task, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML document: %w", err)
	}

	var tasks []Task

	doc.Find("article.hn-tterm-card").Each(func(i int, card *goquery.Selection) {
		task := parseCard(card)
		if task.Title != "" {
			task.ContentHash = task.ComputeHash()
			tasks = append(tasks, task)
		}
	})

	return tasks, nil
}

func parseCard(card *goquery.Selection) Task {
	// Title
	titleElem := card.Find("h3.hn-tterm-title")
	title := cleanText(titleElem.Text())
	title = strings.TrimPrefix(title, ">")
	title = strings.TrimSpace(title)

	// File name (.hn-tterm-file)
	fileName := cleanText(card.Find(".hn-tterm-file").Text())

	// Initiator
	initiator := cleanText(card.Find(`[data-metadata="initiator"] dd`).Text())

	// Campaign
	campaign := cleanText(card.Find(`[data-metadata="campaign"] dd`).Text())

	// Description (.hn-tterm-point)
	pointElem := card.Find(".hn-tterm-point")
	// clone or strip checkmark
	desc := cleanText(pointElem.Text())
	desc = strings.TrimPrefix(desc, "- [x]")
	desc = strings.TrimPrefix(desc, "- [ ]")
	desc = strings.TrimSpace(desc)

	// Prize pool (.hn-tterm-prize .sr-only)
	prizeElem := card.Find(".hn-tterm-prize .sr-only")
	var prize string
	if prizeElem.Length() > 0 {
		prizeText := cleanText(prizeElem.Text())
		prizeText = strings.TrimPrefix(prizeText, "PULA NAGRÓD:")
		prize = strings.TrimSpace(prizeText)
		if prize != "" && !strings.HasSuffix(prize, "PLN") && !strings.HasSuffix(prize, "zł") {
			prize = prize + " PLN"
		}
	} else {
		prize = "Brak informacji"
	}

	// Logo URL
	logoURL, _ := card.Find(".hn-tterm-logo img").Attr("src")
	logoURL = strings.TrimSpace(logoURL)

	// Unique ID
	id := generateTaskID(title, fileName)

	return Task{
		ID:          id,
		Title:       title,
		FileName:    fileName,
		Initiator:   initiator,
		Campaign:    campaign,
		Description: desc,
		Prize:       prize,
		LogoURL:     logoURL,
	}
}

func generateTaskID(title, fileName string) string {
	if fileName != "" {
		base := strings.TrimSuffix(fileName, ".md")
		base = strings.ToLower(strings.TrimSpace(base))
		if base != "" {
			return base
		}
	}
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonAlphaNumericRegex.ReplaceAllString(slug, "")
	if slug == "" {
		slug = "task"
	}
	return slug
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ") // non-breaking space
	s = multipleSpacesRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
