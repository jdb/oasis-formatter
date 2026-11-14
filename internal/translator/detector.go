package translator

import (
	"strings"
	"unicode"
)

// SupportedLanguage describes a selectable/detectable language.
type SupportedLanguage struct {
	Code string `json:"code" doc:"ISO-639-1 language code"`
	Name string `json:"name" doc:"English display name"`
}

// SupportedLanguages lists the primary languages exposed in the UI selector.
var SupportedLanguages = []SupportedLanguage{
	{Code: "en", Name: "English"},
	{Code: "fr", Name: "French"},
	{Code: "es", Name: "Spanish"},
	{Code: "de", Name: "German"},
	{Code: "it", Name: "Italian"},
	{Code: "pt", Name: "Portuguese"},
	{Code: "nl", Name: "Dutch"},
	{Code: "ru", Name: "Russian"},
	{Code: "ja", Name: "Japanese"},
	{Code: "zh", Name: "Chinese"},
	{Code: "pl", Name: "Polish"},
	{Code: "sv", Name: "Swedish"},
	{Code: "ko", Name: "Korean"},
	{Code: "ar", Name: "Arabic"},
}

var languageNames = map[string]string{
	"en":    "English",
	"fr":    "French",
	"es":    "Spanish",
	"de":    "German",
	"it":    "Italian",
	"pt":    "Portuguese",
	"nl":    "Dutch",
	"ru":    "Russian",
	"ja":    "Japanese",
	"zh":    "Chinese",
	"zh-cn": "Chinese (Simplified)",
	"zh-tw": "Chinese (Traditional)",
	"pl":    "Polish",
	"sv":    "Swedish",
	"ko":    "Korean",
	"ar":    "Arabic",
	"da":    "Danish",
	"fi":    "Finnish",
	"no":    "Norwegian",
	"cs":    "Czech",
	"el":    "Greek",
	"tr":    "Turkish",
	"uk":    "Ukrainian",
}

// LanguageName returns the human-readable language name for an ISO code.
func LanguageName(code string) string {
	norm := strings.ToLower(strings.TrimSpace(code))
	if name, ok := languageNames[norm]; ok {
		return name
	}
	if len(norm) >= 2 {
		if name, ok := languageNames[norm[:2]]; ok {
			return name
		}
	}
	if norm == "" {
		return "Unknown"
	}
	return strings.ToUpper(norm)
}

var langStopwords = map[string][]string{
	"fr": {"le", "la", "les", "des", "dans", "une", "est", "pour", "que", "pas", "sur", "plus", "avec", "sont", "mais", "comme", "cette", "tout", "aux", "nous", "vous", "était", "je", "me", "suis", "de", "du", "au", "et", "un", "ne", "se", "mon", "ma", "mes"},
	"en": {"the", "and", "that", "have", "for", "not", "with", "you", "this", "but", "his", "from", "they", "say", "her", "she", "will", "one", "all", "would", "there", "their", "was", "were", "been", "which", "when"},
	"es": {"el", "la", "los", "las", "del", "una", "por", "con", "para", "como", "más", "pero", "sus", "este", "esta", "entre", "cuando", "todo", "también", "fue", "había", "de", "en", "un", "no", "que", "se", "al", "lo", "su", "cuyo", "mucho", "tiempo"},
	"de": {"der", "die", "das", "und", "in", "den", "von", "zu", "mit", "sich", "auf", "für", "nicht", "ist", "dem", "als", "auch", "es", "an", "werden", "aus", "ein", "eine", "dass", "er", "sie", "wurde", "ohne"},
	"it": {"il", "la", "che", "di", "un", "una", "per", "non", "con", "sono", "come", "più", "della", "nella", "questo", "questa", "anche", "stato", "hanno", "suo", "del", "nel", "alla", "degli", "delle"},
	"pt": {"o", "os", "as", "um", "uma", "para", "com", "não", "por", "mais", "como", "mas", "foi", "ao", "ele", "das", "tem", "à", "seu", "sua", "ou", "quando", "dos", "nos", "pelo", "pela"},
	"nl": {"het", "een", "van", "dat", "op", "te", "voor", "niet", "met", "zijn", "ook", "aan", "maar", "om", "door", "over", "bij", "naar", "uit", "heeft", "worden"},
}

// DetectLanguageHeuristic detects the ISO-639-1 language code from a sample text
// using Unicode scripts and stopword frequency.
func DetectLanguageHeuristic(sample string) (code string, name string, confidence float64) {
	var kana, han, hangul, cyrillic, arabic int
	for _, r := range sample {
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			kana++
		case unicode.Is(unicode.Han, r):
			han++
		case unicode.Is(unicode.Hangul, r):
			hangul++
		case unicode.Is(unicode.Cyrillic, r):
			cyrillic++
		case unicode.Is(unicode.Arabic, r):
			arabic++
		}
	}
	if kana > 0 {
		return "ja", LanguageName("ja"), 0.98
	}
	if hangul > 2 {
		return "ko", LanguageName("ko"), 0.98
	}
	if han > 2 {
		return "zh", LanguageName("zh"), 0.96
	}
	if cyrillic > 3 {
		return "ru", LanguageName("ru"), 0.95
	}
	if arabic > 3 {
		return "ar", LanguageName("ar"), 0.95
	}

	lower := strings.ToLower(sample)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if len(words) == 0 {
		return "en", "English", 0.50
	}

	scores := map[string]int{}
	for _, w := range words {
		for lang, stops := range langStopwords {
			for _, sw := range stops {
				if w == sw {
					scores[lang] += 2
				}
			}
		}
	}

	// Characteristic characters boost
	if strings.ContainsAny(lower, "œêèçàù«»") {
		scores["fr"] += 4
	}
	if strings.ContainsAny(lower, "ßäöü") {
		scores["de"] += 4
	}
	if strings.ContainsAny(lower, "ñ¿¡áéíóú") {
		scores["es"] += 4
	}
	if strings.ContainsAny(lower, "ãõç") {
		scores["pt"] += 4
	}

	bestLang := "en"
	bestScore := -1
	for _, lang := range []string{"fr", "en", "es", "de", "it", "pt", "nl"} {
		if scores[lang] > bestScore {
			bestScore = scores[lang]
			bestLang = lang
		}
	}

	conf := 0.75
	if bestScore >= 6 {
		conf = 0.96
	} else if bestScore >= 2 {
		conf = 0.88
	}
	return bestLang, LanguageName(bestLang), conf
}
