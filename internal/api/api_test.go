package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jdb/oasis-formatter/internal/translator"
)

type stubProvider struct{}

func (s *stubProvider) TranslateParagraph(ctx context.Context, text, sourceLang, targetLang string) (translator.Result, error) {
	return translator.Result{
		TranslatedText: "[EN] " + text,
		SourceLang:     "fr",
		SourceLangName: "French",
		Confidence:     0.99,
	}, nil
}

func TestHealthAndOpenAPI(t *testing.T) {
	h := NewHandler(&stubProvider{}, "")

	// GET /api/health
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(rec.Header().Get("X-Request-Id")) != 16 {
		t.Errorf("expected 16-char X-Request-Id, got %q", rec.Header().Get("X-Request-Id"))
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Errorf("unexpected health body: %s", rec.Body.String())
	}

	// GET /openapi.json
	reqOA := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	recOA := httptest.NewRecorder()
	h.ServeHTTP(recOA, reqOA)
	if recOA.Code != http.StatusOK {
		t.Fatalf("expected 200 for /openapi.json, got %d", recOA.Code)
	}
	if !strings.Contains(recOA.Header().Get("Content-Type"), "openapi+json") {
		t.Errorf("unexpected content-type: %s", recOA.Header().Get("Content-Type"))
	}

	// GET /api/nonexistent -> 404 application/problem+json
	req404 := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	rec404 := httptest.NewRecorder()
	h.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec404.Code)
	}
	if !strings.Contains(rec404.Header().Get("Content-Type"), "application/problem+json") {
		t.Errorf("expected problem+json, got %s", rec404.Header().Get("Content-Type"))
	}
}

func TestTranslateStreamSSE(t *testing.T) {
	h := NewHandler(&stubProvider{}, "")

	payload, _ := json.Marshal(TranslateRequest{
		Text:       "Premier paragraphe.\n\nDeuxième paragraphe.",
		TargetLang: "en",
		DelayMs:    -1, // no delay in unit test
	})

	req := httptest.NewRequest(http.MethodPost, "/api/translate/stream", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, expected := range []string{
		"event: paragraphs_split",
		"event: language_detected",
		`"sourceLang":"fr"`,
		"event: paragraph_translated",
		`"[EN] Premier paragraphe."`,
		`"[EN] Deuxième paragraphe."`,
		"event: done",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected SSE stream to contain %q, got:\n%s", expected, body)
		}
	}
}

func TestFetchURLValidationAndHTMLExtraction(t *testing.T) {
	h := NewHandler(&stubProvider{}, "")

	// Reject loopback / private URLs
	payload, _ := json.Marshal(FetchURLRequest{URL: "http://127.0.0.1:8080/api/health"})
	req := httptest.NewRequest(http.MethodPost, "/api/fetch-url", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for loopback URL, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify HTML extraction preserves paragraphs and strips scripts
	rawHTML := `<html><head><style>body{}</style></head><body><script>alert(1)</script><p>First paragraph.</p><p>Second &amp; final paragraph.</p></body></html>`
	got := ExtractReadableText(rawHTML, "text/html; charset=utf-8")
	if got != "First paragraph.\n\nSecond & final paragraph." {
		t.Fatalf("unexpected extracted text: %q", got)
	}
}

func TestExtractEPUBEndpoint(t *testing.T) {
	// Build a minimal valid EPUB archive in memory
	var epubBuf bytes.Buffer
	zw := zip.NewWriter(&epubBuf)

	addFile := func(name, content string) {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}

	addFile("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	addFile("OEBPS/content.opf", `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Sample Oasis Book</dc:title>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`)

	addFile("OEBPS/ch1.xhtml", `<html><body><p>Chapter One paragraph.</p></body></html>`)
	addFile("OEBPS/ch2.xhtml", `<html><body><p>Chapter Two paragraph.</p></body></html>`)
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "book.epub")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = fw.Write(epubBuf.Bytes())
	_ = mw.Close()

	h := NewHandler(&stubProvider{}, "")
	req := httptest.NewRequest(http.MethodPost, "/api/extract-epub", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Text     string `json:"text"`
		Title    string `json:"title"`
		Chapters int    `json:"chapters"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Title != "Sample Oasis Book" {
		t.Errorf("expected title 'Sample Oasis Book', got %q", resp.Title)
	}
	if resp.Chapters != 2 {
		t.Errorf("expected 2 chapters, got %d", resp.Chapters)
	}
	if resp.Text != "Chapter One paragraph.\n\nChapter Two paragraph." {
		t.Errorf("unexpected extracted epub text: %q", resp.Text)
	}
}
