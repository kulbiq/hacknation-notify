package scraper

import (
	"strings"
	"testing"
)

const sampleHTML = `
<!DOCTYPE html>
<html>
<body>
<article class="hn-tterm-card hn-tterm-full" data-complete="true" data-reduced="false">
  <header class="hn-tterm-head">
    <div class="hn-tterm-titlewrap">
      <h3 class="hn-tterm-title"><span aria-hidden="true">&gt; </span>RISK MINER</h3>
      <div class="hn-tterm-rule" aria-hidden="true"></div>
    </div>
    <div class="hn-tterm-logo">
      <img src="https://mudnibfuppwadjkynscc.supabase.co/storage/v1/object/public/task-files/057d36b5-cb92-4062-9393-983cb8a96088/logos/1100979a-744b-4de1-96a3-7551e06d2f30.png" alt="" loading="lazy">
    </div>
  </header>
  <div class="hn-tterm">
    <div class="hn-tterm-bar" aria-hidden="true"><span></span><span></span><span></span></div>
    <div class="hn-tterm-body">
      <p class="hn-tterm-cmd" aria-hidden="true">
        <span class="hn-tterm-prompt">hacknation@tasks:~$ </span>
        <span class="hn-tterm-bin">bat </span>
        <span class="hn-tterm-file">risk_miner.md</span>
      </p>
      <h4 class="hn-tterm-h" data-revealed="true"># O projekcie</h4>
      <dl class="hn-tterm-meta">
        <div class="hn-tterm-meta-row" data-metadata="initiator" data-revealed="true">
          <span class="hn-tterm-separator" aria-hidden="true">|</span>
          <dt>Inicjator projektu</dt>
          <span class="hn-tterm-separator" aria-hidden="true">|</span>
          <dd>MINISTERSTWO FUNDUSZY I POLITYKI REGIONALNEJ</dd>
          <span class="hn-tterm-separator" aria-hidden="true">|</span>
        </div>
        <div class="hn-tterm-meta-row" data-metadata="campaign" data-revealed="true">
          <span class="hn-tterm-separator" aria-hidden="true">|</span>
          <dt>Kampania</dt>
          <span class="hn-tterm-separator" aria-hidden="true">|</span>
          <dd>RISK MINER</dd>
          <span class="hn-tterm-separator" aria-hidden="true">|</span>
        </div>
      </dl>
      <p class="hn-tterm-p hn-tterm-point" data-revealed="true">
        <span class="hn-tterm-check" aria-hidden="true">- [x]</span>
        <span>Każda inwestycja publiczna to wiele decyzji, zależności i potencjalnych zagrożeń. Jak wykorzystać dane, AI i nowoczesne technologie, aby wcześniej identyfikować ryzyka, przewidywać problemy i skuteczniej planować realizację inwestycji?</span>
      </p>
      <div class="hn-tterm-prize" data-revealed="true">
        <span class="sr-only">PULA NAGRÓD: <!-- -->25 000</span>
        <p class="hn-tterm-prize-label" aria-hidden="true">PULA NAGRÓD</p>
      </div>
    </div>
  </div>
</article>

<article class="hn-tterm-card hn-tterm-full">
  <header class="hn-tterm-head">
    <div class="hn-tterm-titlewrap">
      <h3 class="hn-tterm-title">KOMPAS KOMPETENCJI</h3>
    </div>
    <div class="hn-tterm-logo">
      <img src="https://example.com/logo2.png">
    </div>
  </header>
  <div class="hn-tterm">
    <div class="hn-tterm-body">
      <p class="hn-tterm-cmd">
        <span class="hn-tterm-file">kompas_kompetencji.md</span>
      </p>
      <dl class="hn-tterm-meta">
        <div class="hn-tterm-meta-row" data-metadata="initiator">
          <dt>Inicjator projektu</dt>
          <dd>MINISTERSTWO CYFRYZACJI</dd>
        </div>
        <div class="hn-tterm-meta-row" data-metadata="campaign">
          <dt>Kampania</dt>
          <dd>KOMPAS KOMPETENCJI</dd>
        </div>
      </dl>
      <p class="hn-tterm-p hn-tterm-point">
        <span>Opis zadania bez puli nagród.</span>
      </p>
    </div>
  </div>
</article>
</body>
</html>
`

func TestParseHTML(t *testing.T) {
	s := NewScraper("", nil)
	tasks, err := s.ParseHTML(strings.NewReader(sampleHTML))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Task 1
	t1 := tasks[0]
	if t1.Title != "RISK MINER" {
		t.Errorf("expected Title 'RISK MINER', got '%s'", t1.Title)
	}
	if t1.ID != "risk_miner" {
		t.Errorf("expected ID 'risk_miner', got '%s'", t1.ID)
	}
	if t1.FileName != "risk_miner.md" {
		t.Errorf("expected FileName 'risk_miner.md', got '%s'", t1.FileName)
	}
	if t1.Initiator != "MINISTERSTWO FUNDUSZY I POLITYKI REGIONALNEJ" {
		t.Errorf("expected Initiator 'MINISTERSTWO FUNDUSZY I POLITYKI REGIONALNEJ', got '%s'", t1.Initiator)
	}
	if t1.Campaign != "RISK MINER" {
		t.Errorf("expected Campaign 'RISK MINER', got '%s'", t1.Campaign)
	}
	if t1.Prize != "25 000 PLN" {
		t.Errorf("expected Prize '25 000 PLN', got '%s'", t1.Prize)
	}
	if !strings.HasPrefix(t1.Description, "Każda inwestycja publiczna") {
		t.Errorf("expected Description to start with 'Każda inwestycja publiczna', got '%s'", t1.Description)
	}
	if t1.ContentHash == "" {
		t.Errorf("expected non-empty ContentHash")
	}

	// Task 2
	t2 := tasks[1]
	if t2.Title != "KOMPAS KOMPETENCJI" {
		t.Errorf("expected Title 'KOMPAS KOMPETENCJI', got '%s'", t2.Title)
	}
	if t2.Prize != "Brak informacji" {
		t.Errorf("expected Prize 'Brak informacji', got '%s'", t2.Prize)
	}
	if t2.Initiator != "MINISTERSTWO CYFRYZACJI" {
		t.Errorf("expected Initiator 'MINISTERSTWO CYFRYZACJI', got '%s'", t2.Initiator)
	}
}
