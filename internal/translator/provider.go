package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// Provider translates paragraphs and detects source languages.
type Provider interface {
	TranslateParagraph(ctx context.Context, text string, sourceLang string, targetLang string) (Result, error)
}

// GTXProvider uses the Google Translate GTX endpoint with heuristic fallback.
type GTXProvider struct {
	Client  *http.Client
	BaseURL string
}

// NewGTXProvider creates a production translation provider.
func NewGTXProvider() *GTXProvider {
	return &GTXProvider{
		Client: &http.Client{
			Timeout: 12 * time.Second,
		},
		BaseURL: "https://translate.googleapis.com/translate_a/single",
	}
}

// TranslateParagraph translates a single paragraph and returns the detected source language.
func (p *GTXProvider) TranslateParagraph(ctx context.Context, text string, sourceLang string, targetLang string) (Result, error) {
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

	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return p.fallbackTranslate(trimmed, heurCode, heurName, heurConf, tl), nil
	}
	q := u.Query()
	q.Set("client", "gtx")
	q.Set("sl", sl)
	q.Set("tl", tl)
	q.Set("dt", "t")
	q.Set("q", trimmed)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return p.fallbackTranslate(trimmed, heurCode, heurName, heurConf, tl), nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) OasisFormatter/0.1")

	resp, err := p.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return p.fallbackTranslate(trimmed, heurCode, heurName, heurConf, tl), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return p.fallbackTranslate(trimmed, heurCode, heurName, heurConf, tl), nil
	}

	parsed, err := parseGTXResponse(body, heurCode, heurConf)
	if err != nil || strings.TrimSpace(parsed.TranslatedText) == "" {
		return p.fallbackTranslate(trimmed, heurCode, heurName, heurConf, tl), nil
	}
	return parsed, nil
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

func (p *GTXProvider) fallbackTranslate(text, srcCode, srcName string, conf float64, targetLang string) Result {
	if strings.EqualFold(srcCode, targetLang) {
		return Result{
			TranslatedText: text,
			SourceLang:     srcCode,
			SourceLangName: srcName,
			Confidence:     conf,
		}
	}
	return Result{
		TranslatedText: fmt.Sprintf("[%s → %s] %s", strings.ToUpper(srcCode), strings.ToUpper(targetLang), text),
		SourceLang:     srcCode,
		SourceLangName: srcName,
		Confidence:     conf,
	}
}
