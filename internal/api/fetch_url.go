package api

import (
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// FetchURLRequest is the JSON payload for POST /api/fetch-url.
type FetchURLRequest struct {
	URL string `json:"url" example:"https://example.com/story.txt" doc:"HTTP or HTTPS URL of text or webpage content"`
}

// FetchURLOutput is the JSON response for POST /api/fetch-url.
type FetchURLOutput struct {
	Body struct {
		Text        string `json:"text"`
		ContentType string `json:"contentType"`
	}
}

var (
	scriptStyleRe = regexp.MustCompile(`(?is)<(script|style|noscript|svg)[^>]*>.*?</(script|style|noscript|svg)>`)
	blockBreakRe  = regexp.MustCompile(`(?i)</?(p|div|br|h[1-6]|li|blockquote|article|section|tr)[^>]*>`)
	anyTagRe      = regexp.MustCompile(`<[^>]+>`)
	multiSpaceRe  = regexp.MustCompile(`[ \t]+`)
	multiBlankRe  = regexp.MustCompile(`\n{3,}`)
)

func isDisallowedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

func validatePublicHTTPURL(ctx context.Context, rawURL string) (*url.URL, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, fmt.Errorf("URL is required")
	}
	u, err := url.Parse(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("please enter a valid http:// or https:// URL")
	}

	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if isDisallowedIP(ip) {
			return nil, fmt.Errorf("private or local network URLs are not allowed")
		}
		return u, nil
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("could not resolve host %q", host)
	}
	for _, addr := range ips {
		if isDisallowedIP(addr.IP) {
			return nil, fmt.Errorf("private or local network URLs are not allowed")
		}
	}
	return u, nil
}

// ExtractReadableText converts HTML into clean paragraph-separated text,
// or returns plain text as-is.
func ExtractReadableText(raw string, contentType string) string {
	ct := strings.ToLower(contentType)
	if !strings.Contains(ct, "html") {
		return strings.TrimSpace(raw)
	}

	cleaned := scriptStyleRe.ReplaceAllString(raw, "\n")
	cleaned = blockBreakRe.ReplaceAllString(cleaned, "\n\n")
	cleaned = anyTagRe.ReplaceAllString(cleaned, " ")
	cleaned = html.UnescapeString(cleaned)
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	lines := strings.Split(cleaned, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(multiSpaceRe.ReplaceAllString(line, " "))
	}
	joined := strings.Join(lines, "\n")
	joined = multiBlankRe.ReplaceAllString(joined, "\n\n")
	return strings.TrimSpace(joined)
}

func handleFetchURL(ctx context.Context, rawURL string) (string, string, error) {
	parsed, err := validatePublicHTTPURL(ctx, rawURL)
	if err != nil {
		return "", "", huma.Error400BadRequest(err.Error())
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", "", huma.Error400BadRequest("invalid URL")
	}
	req.Header.Set("User-Agent", "OasisFormatter/0.1 (+https://github.com/jdb/oasis-formatter)")
	req.Header.Set("Accept", "text/plain, text/markdown, text/html;q=0.9, */*;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", huma.Error502BadGateway("failed to fetch URL: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", huma.Error502BadGateway(fmt.Sprintf("remote server returned HTTP %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", "", huma.Error502BadGateway("failed to read remote content")
	}

	contentType := resp.Header.Get("Content-Type")
	text := ExtractReadableText(string(bodyBytes), contentType)
	if text == "" {
		return "", "", huma.Error422UnprocessableEntity("no readable text found at that URL")
	}

	return text, contentType, nil
}
