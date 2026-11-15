package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Result holds the translation of a single paragraph along with detected language metadata.
type Result struct {
	TranslatedText string
	SourceLang     string
	SourceLangName string
	Confidence     float64
}

// StatusUpdate communicates progress, rate-limit pauses, and retry wait times to the caller.
type StatusUpdate struct {
	State      string // "translating" | "paused"
	Message    string
	RetryInSec int
}

// Provider translates paragraphs and detects source languages.
type Provider interface {
	TranslateParagraph(ctx context.Context, text string, sourceLang string, targetLang string) (Result, error)
}

// BatchProvider is an optional interface for providers that can translate multiple paragraphs
// in a single request and report rate-limit pause/retry status without dropping translations.
type BatchProvider interface {
	Provider
	TranslateBatch(ctx context.Context, paragraphs []string, sourceLang string, targetLang string, onStatus func(StatusUpdate)) ([]Result, error)
}

// GTXProvider uses the Google Translate GTX endpoint with batching and patient retry/backoff
// (never degrading or dropping translations on rate limits).
type GTXProvider struct {
	Client  *http.Client
	BaseURL string
	// MaxBackoff caps how long a single retry pause can wait (default 2 minutes).
	MaxBackoff time.Duration
	// BaseBackoff sets the initial pause on rate limit (default 10 seconds).
	BaseBackoff time.Duration
}

// NewGTXProvider creates a production translation provider.
func NewGTXProvider() *GTXProvider {
	return &GTXProvider{
		Client: &http.Client{
			Timeout: 25 * time.Second,
		},
		BaseURL:     "https://translate.googleapis.com/translate_a/single",
		BaseBackoff: 10 * time.Second,
		MaxBackoff:  120 * time.Second,
	}
}

// TranslateParagraph translates a single paragraph with retry/backoff (never dropping translation).
func (p *GTXProvider) TranslateParagraph(ctx context.Context, text string, sourceLang string, targetLang string) (Result, error) {
	results, err := p.TranslateBatch(ctx, []string{text}, sourceLang, targetLang, nil)
	if err != nil {
		return Result{}, err
	}
	if len(results) == 0 {
		return Result{}, fmt.Errorf("empty translation result")
	}
	return results[0], nil
}

var batchMarkerRe = regexp.MustCompile(`\s*§§§(\d+)§§§\s*`)

// TranslateBatch translates a slice of paragraphs in as few API calls as possible,
// splitting if needed and pausing/retrying indefinitely (until ctx is canceled) on rate limits.
func (p *GTXProvider) TranslateBatch(
	ctx context.Context,
	paragraphs []string,
	sourceLang string,
	targetLang string,
	onStatus func(StatusUpdate),
) ([]Result, error) {
	if len(paragraphs) == 0 {
		return nil, nil
	}
	if len(paragraphs) == 1 {
		res, err := p.translateSingleWithRetry(ctx, paragraphs[0], sourceLang, targetLang, onStatus)
		if err != nil {
			return nil, err
		}
		return []Result{res}, nil
	}

	var b strings.Builder
	for i, para := range paragraphs {
		if i > 0 {
			fmt.Fprintf(&b, "\n\n§§§%d§§§\n\n", i)
		}
		b.WriteString(strings.TrimSpace(para))
	}

	combinedRes, err := p.translateSingleWithRetry(ctx, b.String(), sourceLang, targetLang, onStatus)
	if err != nil {
		return nil, err
	}

	// Verify all expected markers survived translation in order
	matches := batchMarkerRe.FindAllStringSubmatchIndex(combinedRes.TranslatedText, -1)
	if len(matches) == len(paragraphs)-1 {
		out := make([]Result, len(paragraphs))
		prevEnd := 0
		valid := true
		for idx, m := range matches {
			markerNum, _ := strconv.Atoi(combinedRes.TranslatedText[m[2]:m[3]])
			if markerNum != idx+1 {
				valid = false
				break
			}
			out[idx] = Result{
				TranslatedText: strings.TrimSpace(combinedRes.TranslatedText[prevEnd:m[0]]),
				SourceLang:     combinedRes.SourceLang,
				SourceLangName: combinedRes.SourceLangName,
				Confidence:     combinedRes.Confidence,
			}
			prevEnd = m[1]
		}
		if valid {
			out[len(paragraphs)-1] = Result{
				TranslatedText: strings.TrimSpace(combinedRes.TranslatedText[prevEnd:]),
				SourceLang:     combinedRes.SourceLang,
				SourceLangName: combinedRes.SourceLangName,
				Confidence:     combinedRes.Confidence,
			}
			return out, nil
		}
	}

	// If a marker was altered by the translator, split the batch in half and retry
	mid := len(paragraphs) / 2
	left, err := p.TranslateBatch(ctx, paragraphs[:mid], sourceLang, targetLang, onStatus)
	if err != nil {
		return nil, err
	}
	right, err := p.TranslateBatch(ctx, paragraphs[mid:], sourceLang, targetLang, onStatus)
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

func (p *GTXProvider) translateSingleWithRetry(
	ctx context.Context,
	text string,
	sourceLang string,
	targetLang string,
	onStatus func(StatusUpdate),
) (Result, error) {
	trimmed := strings.TrimSpace(text)
	heurCode, heurName, heurConf := DetectLanguageHeuristic(trimmed)
	if trimmed == "" {
		return Result{
			TranslatedText: "",
			SourceLang:     heurCode,
			SourceLangName: heurName,
			Confidence:     heurConf,
		}, nil
	}

	sl := sourceLang
	if sl == "" {
		sl = "auto"
	}
	tl := strings.ToLower(strings.TrimSpace(targetLang))
	if tl == "" {
		tl = "en"
	}

	baseBackoff := p.BaseBackoff
	if baseBackoff <= 0 {
		baseBackoff = 10 * time.Second
	}
	maxBackoff := p.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 120 * time.Second
	}

	attempt := 0
	for {
		res, retryWait, err := p.doGTXRequest(ctx, trimmed, sl, tl, heurCode, heurConf)
		if err == nil {
			return res, nil
		}

		// If caller canceled, stop immediately.
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}

		attempt++
		wait := baseBackoff * time.Duration(1<<min(attempt-1, 4))
		if retryWait > wait {
			wait = retryWait
		}
		if wait > maxBackoff {
			wait = maxBackoff
		}

		waitSec := int(wait.Seconds())
		if waitSec < 1 {
			waitSec = 1
		}

		if onStatus != nil {
			onStatus(StatusUpdate{
				State:      "paused",
				RetryInSec: waitSec,
				Message:    fmt.Sprintf("Translation API throttled (%v) — pausing %ds before retrying (attempt %d). No paragraphs will be dropped.", err, waitSec, attempt),
			})
		}

		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (p *GTXProvider) doGTXRequest(
	ctx context.Context,
	text string,
	sl string,
	tl string,
	heurCode string,
	heurConf float64,
) (Result, time.Duration, error) {
	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return Result{}, 0, err
	}
	q := u.Query()
	q.Set("client", "gtx")
	q.Set("sl", sl)
	q.Set("tl", tl)
	q.Set("dt", "t")
	u.RawQuery = q.Encode()

	form := url.Values{}
	form.Set("q", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return Result{}, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) OasisFormatter/0.1")

	resp, err := p.Client.Do(req)
	if err != nil {
		return Result{}, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		var retryAfter time.Duration
		if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
			if secs, perr := strconv.Atoi(ra); perr == nil && secs > 0 {
				retryAfter = time.Duration(secs) * time.Second
			}
		}
		return Result{}, retryAfter, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return Result{}, 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Result{}, 0, err
	}

	parsed, err := parseGTXResponse(body, heurCode, heurConf)
	if err != nil {
		return Result{}, 0, err
	}
	return parsed, 0, nil
}

func parseGTXResponse(body []byte, fallbackLang string, fallbackConf float64) (Result, error) {
	var raw []any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Result{}, err
	}
	if len(raw) == 0 {
		return Result{}, fmt.Errorf("empty gtx payload")
	}

	var b strings.Builder
	if sentences, ok := raw[0].([]any); ok {
		for _, s := range sentences {
			if seg, ok := s.([]any); ok && len(seg) > 0 {
				if part, ok := seg[0].(string); ok {
					b.WriteString(part)
				}
			}
		}
	}

	detectedLang := fallbackLang
	if len(raw) > 2 {
		if dl, ok := raw[2].(string); ok && strings.TrimSpace(dl) != "" {
			detectedLang = strings.ToLower(strings.TrimSpace(dl))
		}
	}

	conf := fallbackConf
	if len(raw) > 6 {
		if c, ok := raw[6].(float64); ok && c > 0 {
			conf = c
		}
	}

	return Result{
		TranslatedText: strings.TrimSpace(b.String()),
		SourceLang:     detectedLang,
		SourceLangName: LanguageName(detectedLang),
		Confidence:     conf,
	}, nil
}
