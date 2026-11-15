package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jdb/oasis-formatter/internal/translator"
)

// HealthOutput represents the GET /api/health response.
type HealthOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"Health status"`
	}
}

// LanguagesOutput represents the GET /api/languages response.
type LanguagesOutput struct {
	Body struct {
		Languages []translator.SupportedLanguage `json:"languages"`
	}
}

// TranslateRequest is the JSON payload for POST /api/translate/stream.
type TranslateRequest struct {
	Text       string   `json:"text,omitempty" doc:"Raw text to split into paragraphs and translate"`
	Paragraphs []string `json:"paragraphs,omitempty" doc:"Pre-split paragraphs to translate"`
	TargetLang string   `json:"targetLang" example:"en" doc:"Target ISO-639-1 language code"`
	DelayMs    int      `json:"delayMs,omitempty" example:"160" doc:"Pacing delay in milliseconds between streamed paragraphs"`
}

// TranslateBatchOutput is the JSON response for POST /api/translate (non-streaming helper).
type TranslateBatchOutput struct {
	Body struct {
		SourceLang     string              `json:"sourceLang"`
		SourceLangName string              `json:"sourceLangName"`
		Confidence     float64             `json:"confidence"`
		TargetLang     string              `json:"targetLang"`
		TargetLangName string              `json:"targetLangName"`
		Paragraphs     []TranslatedPairDTO `json:"paragraphs"`
	}
}

// TranslatedPairDTO represents a single source + translation paragraph pair.
type TranslatedPairDTO struct {
	Index       int    `json:"index"`
	Source      string `json:"source"`
	Translation string `json:"translation"`
}

// NewHandler wires the Huma v2 OpenAPI 3.1 router, SSE translation stream endpoint,
// and the SPA static file handler (with mespousses.fr caching semantics).
func NewHandler(provider translator.Provider, staticDir string) http.Handler {
	if provider == nil {
		provider = translator.NewGTXProvider()
	}

	mux := http.NewServeMux()
	cfg := huma.DefaultConfig("oasis-formatter", "0.1.0")
	cfg.CreateHooks = nil
	cfg.DocsPath = "/api/docs"
	humaAPI := humago.New(mux, cfg)

	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/api/health",
		Summary:     "Health check",
	}, func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
		out := &HealthOutput{}
		out.Body.Status = "ok"
		return out, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "list-languages",
		Method:      http.MethodGet,
		Path:        "/api/languages",
		Summary:     "List supported destination languages",
	}, func(ctx context.Context, input *struct{}) (*LanguagesOutput, error) {
		out := &LanguagesOutput{}
		out.Body.Languages = translator.SupportedLanguages
		return out, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "translate-batch",
		Method:      http.MethodPost,
		Path:        "/api/translate",
		Summary:     "Split text, detect source language, and translate paragraphs",
	}, func(ctx context.Context, input *struct {
		Body TranslateRequest
	}) (*TranslateBatchOutput, error) {
		paragraphs := input.Body.Paragraphs
		if len(paragraphs) == 0 {
			paragraphs = translator.SplitParagraphs(input.Body.Text)
		}
		targetLang := strings.ToLower(strings.TrimSpace(input.Body.TargetLang))
		if targetLang == "" {
			targetLang = "en"
		}

		sample := strings.Join(paragraphs, "\n\n")
		srcLang, srcName, conf := translator.DetectLanguageHeuristic(sample)

		pairs := make([]TranslatedPairDTO, 0, len(paragraphs))
		detectedFromProvider := false
		for idx, p := range paragraphs {
			res, err := provider.TranslateParagraph(ctx, p, "auto", targetLang)
			if err != nil {
				return nil, huma.Error502BadGateway("translation failed: " + err.Error())
			}
			if !detectedFromProvider && res.SourceLang != "" {
				srcLang = res.SourceLang
				srcName = res.SourceLangName
				conf = res.Confidence
				detectedFromProvider = true
			}
			pairs = append(pairs, TranslatedPairDTO{
				Index:       idx,
				Source:      p,
				Translation: res.TranslatedText,
			})
		}

		out := &TranslateBatchOutput{}
		out.Body.SourceLang = srcLang
		out.Body.SourceLangName = srcName
		out.Body.Confidence = conf
		out.Body.TargetLang = targetLang
		out.Body.TargetLangName = translator.LanguageName(targetLang)
		out.Body.Paragraphs = pairs
		return out, nil
	})

	huma.Register(humaAPI, huma.Operation{
		OperationID: "fetch-url",
		Method:      http.MethodPost,
		Path:        "/api/fetch-url",
		Summary:     "Fetch text or readable article content from a public URL",
	}, func(ctx context.Context, input *struct {
		Body FetchURLRequest
	}) (*FetchURLOutput, error) {
		text, ct, err := handleFetchURL(ctx, input.Body.URL)
		if err != nil {
			return nil, err
		}
		out := &FetchURLOutput{}
		out.Body.Text = text
		out.Body.ContentType = ct
		return out, nil
	})

	// SSE Streaming endpoint: POST /api/translate/stream
	mux.HandleFunc("POST /api/translate/stream", func(w http.ResponseWriter, r *http.Request) {
		handleTranslateStream(w, r, provider)
	})

	// Top-level router that separates API routes, OpenAPI routes, and SPA static assets.
	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/openapi") || path == "/api/health" || path == "/api/languages" || path == "/api/translate" || path == "/api/translate/stream" || path == "/api/fetch-url" || strings.HasPrefix(path, "/api/docs") {
			mux.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "/api/") {
			writeProblemJSON(w, http.StatusNotFound, "not_found", "Not Found", path, "")
			return
		}
		serveSPA(w, r, staticDir)
	})

	return WrapMiddleware(root)
}

func handleTranslateStream(w http.ResponseWriter, r *http.Request, provider translator.Provider) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProblemJSON(w, http.StatusInternalServerError, "streaming_unsupported", "Streaming Unsupported", r.URL.Path, "ResponseWriter does not support flushing")
		return
	}

	var req TranslateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
		writeProblemJSON(w, http.StatusBadRequest, "invalid_request", "Bad Request", r.URL.Path, "Invalid JSON payload")
		return
	}

	paragraphs := req.Paragraphs
	if len(paragraphs) == 0 {
		paragraphs = translator.SplitParagraphs(req.Text)
	} else {
		cleaned := make([]string, 0, len(paragraphs))
		for _, p := range paragraphs {
			if t := strings.TrimSpace(p); t != "" {
				cleaned = append(cleaned, t)
			}
		}
		paragraphs = cleaned
	}

	if len(paragraphs) == 0 {
		writeProblemJSON(w, http.StatusBadRequest, "empty_text", "Empty Text", r.URL.Path, "No paragraphs found to translate")
		return
	}

	targetLang := strings.ToLower(strings.TrimSpace(req.TargetLang))
	if targetLang == "" {
		targetLang = "en"
	}
	targetLangName := translator.LanguageName(targetLang)

	delay := time.Duration(req.DelayMs) * time.Millisecond
	if req.DelayMs < 0 {
		delay = 0
	} else if req.DelayMs == 0 {
		delay = 140 * time.Millisecond
	} else if delay > 2*time.Second {
		delay = 2 * time.Second
	}

	// Set SSE headers for Fly.io & Cloudflare unbuffered streaming
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// 1. Emit initial segmented paragraphs list so the UI can show total paragraph count immediately
	writeSSE(w, "paragraphs_split", map[string]any{
		"total":      len(paragraphs),
		"paragraphs": paragraphs,
	})
	flusher.Flush()

	// 2. Detect language from first paragraph(s) heuristic immediately, and refine on first provider response
	sampleCount := 3
	if len(paragraphs) < sampleCount {
		sampleCount = len(paragraphs)
	}
	heurCode, heurName, heurConf := translator.DetectLanguageHeuristic(strings.Join(paragraphs[:sampleCount], " "))

	ctx := r.Context()
	detectedCode := heurCode
	detectedName := heurName
	detectedConf := heurConf

	for i, srcText := range paragraphs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := provider.TranslateParagraph(ctx, srcText, "auto", targetLang)
		if err != nil {
			writeSSE(w, "paragraph_error", map[string]any{
				"index":   i,
				"message": err.Error(),
			})
			flusher.Flush()
			continue
		}

		if i == 0 {
			if res.SourceLang != "" {
				detectedCode = res.SourceLang
				detectedName = res.SourceLangName
				detectedConf = res.Confidence
			}
			writeSSE(w, "language_detected", map[string]any{
				"sourceLang":     detectedCode,
				"sourceLangName": detectedName,
				"confidence":     detectedConf,
				"targetLang":     targetLang,
				"targetLangName": targetLangName,
			})
			flusher.Flush()
		}

		writeSSE(w, "paragraph_translated", map[string]any{
			"index":          i,
			"source":         srcText,
			"translation":    res.TranslatedText,
			"sourceLang":     detectedCode,
			"sourceLangName": detectedName,
			"targetLang":     targetLang,
			"targetLangName": targetLangName,
		})
		flusher.Flush()

		if i < len(paragraphs)-1 && delay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}

	writeSSE(w, "done", map[string]any{
		"totalTranslated": len(paragraphs),
		"sourceLang":      detectedCode,
		"sourceLangName":  detectedName,
		"targetLang":      targetLang,
		"targetLangName":  targetLangName,
	})
	flusher.Flush()
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b))
}

func serveSPA(w http.ResponseWriter, r *http.Request, staticDir string) {
	if staticDir == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("oasis-formatter API running"))
		return
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "." || cleanPath == "" {
		cleanPath = "index.html"
	}

	fullPath := filepath.Join(staticDir, cleanPath)
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		// SPA fallback to index.html
		fullPath = filepath.Join(staticDir, "index.html")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, fullPath)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/fonts/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
	} else if cleanPath == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}

	http.ServeFile(w, r, fullPath)
}
