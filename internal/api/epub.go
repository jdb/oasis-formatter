package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
)

type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	Metadata struct {
		Titles []string `xml:"title"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

// ExtractTextFromEPUB reads an EPUB ZIP archive and extracts chapter text in spine reading order.
func ExtractTextFromEPUB(r io.ReaderAt, size int64) (text string, title string, chapters int, err error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid EPUB archive: %w", err)
	}

	filesByPath := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		clean := path.Clean(strings.TrimPrefix(f.Name, "/"))
		filesByPath[clean] = f
	}

	readZipEntry := func(p string) ([]byte, error) {
		clean := path.Clean(strings.TrimPrefix(p, "/"))
		f, ok := filesByPath[clean]
		if !ok {
			// Try URL-unescaped path in case hrefs contain %20 etc.
			if unesc, uerr := url.PathUnescape(clean); uerr == nil {
				f, ok = filesByPath[path.Clean(unesc)]
			}
		}
		if !ok {
			return nil, fmt.Errorf("entry %q not found", p)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, 10<<20))
	}

	var orderedPaths []string

	if containerBytes, cerr := readZipEntry("META-INF/container.xml"); cerr == nil {
		var container containerXML
		if xerr := xml.Unmarshal(containerBytes, &container); xerr == nil && len(container.Rootfiles) > 0 {
			opfPath := path.Clean(container.Rootfiles[0].FullPath)
			if opfBytes, oerr := readZipEntry(opfPath); oerr == nil {
				var pkg opfPackage
				if perr := xml.Unmarshal(opfBytes, &pkg); perr == nil {
					if len(pkg.Metadata.Titles) > 0 {
						title = strings.TrimSpace(pkg.Metadata.Titles[0])
					}

					manifestMap := make(map[string]string, len(pkg.Manifest.Items))
					manifestTypes := make(map[string]string, len(pkg.Manifest.Items))
					for _, item := range pkg.Manifest.Items {
						manifestMap[item.ID] = item.Href
						manifestTypes[item.ID] = item.MediaType
					}

					opfDir := path.Dir(opfPath)
					for _, ref := range pkg.Spine.ItemRefs {
						href, ok := manifestMap[ref.IDRef]
						if !ok {
							continue
						}
						mt := strings.ToLower(manifestTypes[ref.IDRef])
						if mt != "" && !strings.Contains(mt, "html") && !strings.Contains(mt, "xml") {
							continue
						}
						resolved := href
						if opfDir != "." && opfDir != "" {
							resolved = path.Join(opfDir, href)
						}
						orderedPaths = append(orderedPaths, path.Clean(resolved))
					}
				}
			}
		}
	}

	// Fallback if spine was empty or missing: scan HTML/XHTML files in sorted order
	if len(orderedPaths) == 0 {
		for p := range filesByPath {
			lower := strings.ToLower(p)
			if strings.HasSuffix(lower, ".xhtml") || strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
				if !strings.Contains(lower, "nav.") && !strings.Contains(lower, "toc.") {
					orderedPaths = append(orderedPaths, p)
				}
			}
		}
		sort.Strings(orderedPaths)
	}

	var sections []string
	for _, p := range orderedPaths {
		rawBytes, rerr := readZipEntry(p)
		if rerr != nil {
			continue
		}
		extracted := ExtractReadableText(string(rawBytes), "text/html")
		if extracted != "" {
			sections = append(sections, extracted)
		}
	}

	if len(sections) == 0 {
		return "", title, 0, fmt.Errorf("no readable text chapters found in EPUB")
	}

	return strings.Join(sections, "\n\n"), title, len(sections), nil
}

func handleExtractEPUB(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		writeProblemJSON(w, http.StatusBadRequest, "invalid_upload", "Bad Request", r.URL.Path, "Could not parse multipart EPUB upload (max 25 MB)")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeProblemJSON(w, http.StatusBadRequest, "missing_file", "Bad Request", r.URL.Path, "Missing 'file' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 25<<20))
	if err != nil {
		writeProblemJSON(w, http.StatusBadRequest, "read_failed", "Bad Request", r.URL.Path, "Failed to read uploaded EPUB file")
		return
	}

	text, title, chapters, err := ExtractTextFromEPUB(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		writeProblemJSON(w, http.StatusUnprocessableEntity, "invalid_epub", "Unprocessable Entity", r.URL.Path, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"text":     text,
		"title":    title,
		"chapters": chapters,
	})
}
