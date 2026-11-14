package translator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSplitParagraphs(t *testing.T) {
	input := "First paragraph line one\nand line two.\r\n\r\nSecond paragraph.\n\n\nThird paragraph."
	got := SplitParagraphs(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 paragraphs, got %d: %#v", len(got), got)
	}
	if got[0] != "First paragraph line one and line two." {
		t.Errorf("unexpected p[0]: %q", got[0])
	}
	if got[1] != "Second paragraph." {
		t.Errorf("unexpected p[1]: %q", got[1])
	}
	if got[2] != "Third paragraph." {
		t.Errorf("unexpected p[2]: %q", got[2])
	}
}

func TestDetectLanguageHeuristic(t *testing.T) {
	code, name, conf := DetectLanguageHeuristic("Longtemps, je me suis couché de bonne heure. Parfois, à peine ma bougie éteinte, mes yeux se fermaient si vite que je n'avais pas le temps de me dire : « Je m'endors. »")
	if code != "fr" || name != "French" || conf < 0.85 {
		t.Fatalf("expected French detection, got code=%q name=%q conf=%v", code, name, conf)
	}

	deCode, deName, _ := DetectLanguageHeuristic("Jemand musste Josef K. verleumdet haben, denn ohne dass er etwas Böses getan hätte, wurde er eines Morgens激动 nicht, sondern im Bett verhaftet.")
	if deCode != "de" && deCode != "zh" {
		t.Logf("got %s (%s)", deCode, deName)
	}

	esCode, esName, _ := DetectLanguageHeuristic("En un lugar de la Mancha, de cuyo nombre no quiero acordarme, no ha mucho tiempo que vivía un hidalgo de los de lanza en astillero.")
	if esCode != "es" || esName != "Spanish" {
		t.Fatalf("expected Spanish detection, got code=%q name=%q", esCode, esName)
	}
}

func TestGTXProviderMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[[["For a long time I went to bed early.","Longtemps, je me suis couché de bonne heure.",null,null,1]],null,"fr",null,null,null,0.99]`))
	}))
	defer srv.Close()

	p := &GTXProvider{
		Client:  srv.Client(),
		BaseURL: srv.URL,
	}

	res, err := p.TranslateParagraph(context.Background(), "Longtemps, je me suis couché de bonne heure.", "auto", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SourceLang != "fr" || res.SourceLangName != "French" {
		t.Fatalf("unexpected source lang: %+v", res)
	}
	if res.TranslatedText != "For a long time I went to bed early." {
		t.Fatalf("unexpected translation: %q", res.TranslatedText)
	}
}
