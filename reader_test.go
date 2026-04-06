package epub

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestOpen_Integration(t *testing.T) {
	files := []string{
		"testdata/gift-of-the-magi.epub",
		"testdata/importance-of-being-earnest.epub",
		"testdata/modest-proposal.epub",
		"testdata/wuthering-heights.epub",
		"testdata/yellow-wallpaper.epub",
	}

	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			r, err := Open(path)
			if err != nil {
				t.Fatalf("Open(%q): %v", path, err)
			}
			defer r.Close()

			if r.Package == nil {
				t.Fatal("Package is nil")
			}
			if len(r.Package.Manifest) == 0 {
				t.Fatal("Manifest is empty")
			}
		})
	}
}

func TestOpen_Error(t *testing.T) {
	_, err := Open("testdata/nonexistent.epub")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadItem_Integration(t *testing.T) {
	files := []string{
		"testdata/gift-of-the-magi.epub",
		"testdata/importance-of-being-earnest.epub",
		"testdata/modest-proposal.epub",
		"testdata/wuthering-heights.epub",
		"testdata/yellow-wallpaper.epub",
	}

	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			r, err := Open(path)
			if err != nil {
				t.Fatalf("Open(%q): %v", path, err)
			}
			defer r.Close()

			for _, item := range r.Package.Manifest {
				data, err := r.ReadItem(item)
				if err != nil {
					t.Errorf("ReadItem(%q): %v", item.ID, err)
					continue
				}
				if len(data) == 0 {
					t.Errorf("ReadItem(%q): returned 0 bytes", item.ID)
				}
			}
		})
	}
}

func TestReadItem_HTMLContent(t *testing.T) {
	r, err := Open("testdata/gift-of-the-magi.epub")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	// Find an XHTML content item.
	var found bool
	for _, item := range r.Package.Manifest {
		if item.MediaType != "application/xhtml+xml" {
			continue
		}
		data, err := r.ReadItem(item)
		if err != nil {
			t.Fatalf("ReadItem(%q): %v", item.ID, err)
		}
		if len(data) == 0 {
			t.Fatalf("ReadItem(%q): 0 bytes", item.ID)
		}
		// XHTML content should start with XML/HTML markup.
		s := strings.TrimSpace(string(data))
		if !strings.HasPrefix(s, "<") && !strings.HasPrefix(s, "<?") {
			t.Errorf("ReadItem(%q): expected HTML/XML content, got %q...", item.ID, s[:min(40, len(s))])
		}
		found = true
		break
	}
	if !found {
		t.Skip("no application/xhtml+xml item in manifest")
	}
}

func TestFirstParagraph_Integration(t *testing.T) {
	books := []string{
		"testdata/gift-of-the-magi.epub",
		"testdata/importance-of-being-earnest.epub",
		"testdata/modest-proposal.epub",
		"testdata/wuthering-heights.epub",
		"testdata/yellow-wallpaper.epub",
	}

	for _, path := range books {
		t.Run(path, func(t *testing.T) {
			r, err := Open(path)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer r.Close()

			byID := make(map[string]Item, len(r.Package.Manifest))
			for _, item := range r.Package.Manifest {
				byID[item.ID] = item
			}

			const maxPages = 5
			var paras []string
			for _, si := range r.Package.Spine {
				if len(paras) == maxPages {
					break
				}
				item, ok := byID[si.IDRef]
				if !ok || item.MediaType != "application/xhtml+xml" {
					continue
				}
				data, err := r.ReadItem(item)
				if err != nil {
					t.Fatalf("ReadItem(%q): %v", item.ID, err)
				}
				if para := firstParagraph(data); para != "" {
					paras = append(paras, para)
				}
			}

			if len(paras) == 0 {
				t.Fatal("no paragraphs found")
			}
			t.Logf("%s\n%s", r.Package.Metadata.Title, strings.Join(paras, "\n"))
		})
	}
}

// firstParagraph scans XHTML and returns the text of the first non-empty <p>.
func firstParagraph(data []byte) string {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	var depth int
	var buf strings.Builder

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "p" {
				depth++
				buf.Reset()
			} else if depth > 0 {
				depth++
			}
		case xml.EndElement:
			if depth > 0 {
				depth--
				if depth == 0 {
					text := strings.Join(strings.Fields(buf.String()), " ")
					if text != "" {
						return text
					}
				}
			}
		case xml.CharData:
			if depth > 0 {
				buf.Write(t)
			}
		}
	}
	return ""
}

func TestReadItem_MissingHref(t *testing.T) {
	r, err := Open("testdata/gift-of-the-magi.epub")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	_, err = r.ReadItem(Item{ID: "fake", Href: "OEBPS/does-not-exist.xhtml"})
	if err == nil {
		t.Fatal("expected error for missing item, got nil")
	}
}
