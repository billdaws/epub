package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

// NavPoint is one entry in the table of contents. Children represent
// nested sections.
type NavPoint struct {
	Title    string     `json:"title"`              // display text for the TOC entry
	Src      string     `json:"src"`                // ZIP-relative path, may include a fragment (e.g. "OEBPS/ch1.xhtml#sec1")
	Children []NavPoint `json:"children,omitempty"` // nested entries, nil if there are none
}

// OpenTOC opens the .epub file at epubPath and returns the table of contents.
// It prefers the EPUB 3 navigation document when present and falls back to
// the EPUB 2 NCX.
func OpenTOC(epubPath string) ([]NavPoint, error) {
	zr, err := zip.OpenReader(epubPath)
	if err != nil {
		return nil, fmt.Errorf("epub: open: %w", err)
	}
	defer zr.Close()

	c, err := parseContainer(&zr.Reader)
	if err != nil {
		return nil, err
	}

	pkg, err := parsePackage(&zr.Reader, c.RootfilePath)
	if err != nil {
		return nil, err
	}

	// Prefer EPUB 3 nav document (properties contains "nav"), fall back to NCX.
	var navItem, ncxItem *Item
	for i := range pkg.Manifest {
		item := &pkg.Manifest[i]
		switch {
		case containsWord(item.Properties, "nav") && navItem == nil:
			navItem = item
		case item.MediaType == "application/x-dtbncx+xml" && ncxItem == nil:
			ncxItem = item
		}
	}

	tocItem := navItem
	if tocItem == nil {
		tocItem = ncxItem
	}
	if tocItem == nil {
		return nil, &MissingTOCError{}
	}

	f := findFile(&zr.Reader, tocItem.Href)
	if f == nil {
		return nil, &FileNotFoundError{Path: tocItem.Href}
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("epub: open TOC: %w", err)
	}
	defer rc.Close()

	tocDir := path.Dir(tocItem.Href)
	if tocItem.MediaType == "application/x-dtbncx+xml" {
		return decodeNCX(rc, tocDir)
	}
	return decodeNav(rc, tocDir)
}

// containsWord reports whether the space-separated s contains word w.
func containsWord(s, w string) bool {
	for _, f := range strings.Fields(s) {
		if f == w {
			return true
		}
	}
	return false
}

// resolveRef prepends dir to ref when dir is non-empty and non-trivial.
func resolveRef(ref, dir string) string {
	if dir == "" || dir == "." || ref == "" {
		return ref
	}
	return dir + "/" + ref
}

// ── NCX (EPUB 2) ─────────────────────────────────────────────────────────────

type xmlNavPoint struct {
	Label   string `xml:"http://www.daisy.org/z3986/2005/ncx/ navLabel>text"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"http://www.daisy.org/z3986/2005/ncx/ content"`
	Children []xmlNavPoint `xml:"http://www.daisy.org/z3986/2005/ncx/ navPoint"`
}

func decodeNCX(r io.Reader, dir string) ([]NavPoint, error) {
	var ncx struct {
		NavPoints []xmlNavPoint `xml:"http://www.daisy.org/z3986/2005/ncx/ navMap>navPoint"`
	}
	if err := xml.NewDecoder(r).Decode(&ncx); err != nil {
		return nil, fmt.Errorf("epub: decode NCX: %w", err)
	}
	return convertNavPoints(ncx.NavPoints, dir), nil
}

func convertNavPoints(xps []xmlNavPoint, dir string) []NavPoint {
	if len(xps) == 0 {
		return nil
	}
	out := make([]NavPoint, len(xps))
	for i, xp := range xps {
		out[i] = NavPoint{
			Title:    strings.TrimSpace(xp.Label),
			Src:      resolveRef(xp.Content.Src, dir),
			Children: convertNavPoints(xp.Children, dir),
		}
	}
	return out
}

// ── Nav document (EPUB 3) ─────────────────────────────────────────────────────

func decodeNav(r io.Reader, dir string) ([]NavPoint, error) {
	dec := xml.NewDecoder(r)

	// Scan for the <nav> element whose epub:type attribute contains "toc".
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("epub: decode nav: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "nav" {
			continue
		}
		for _, attr := range se.Attr {
			if attr.Name.Local == "type" && containsWord(attr.Value, "toc") {
				return parseNavOl(dec, dir)
			}
		}
	}
	return nil, &MissingNavElementError{}
}

// parseNavOl advances past tokens until it hits an <ol>, then delegates.
func parseNavOl(dec *xml.Decoder, dir string) ([]NavPoint, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("epub: nav: expected ol: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "ol" {
				return parseNavList(dec, dir)
			}
			if err := dec.Skip(); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if t.Name.Local == "nav" {
				return nil, nil
			}
		}
	}
}

func parseNavList(dec *xml.Decoder, dir string) ([]NavPoint, error) {
	var points []NavPoint
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("epub: nav ol: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "li" {
				np, err := parseNavLi(dec, dir)
				if err != nil {
					return nil, err
				}
				points = append(points, np)
			} else {
				if err := dec.Skip(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "ol" {
				return points, nil
			}
		}
	}
}

func parseNavLi(dec *xml.Decoder, dir string) (NavPoint, error) {
	var np NavPoint
	for {
		tok, err := dec.Token()
		if err != nil {
			return NavPoint{}, fmt.Errorf("epub: nav li: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "a":
				for _, attr := range t.Attr {
					if attr.Name.Local == "href" {
						np.Src = resolveRef(attr.Value, dir)
					}
				}
				title, err := readText(dec, "a")
				if err != nil {
					return NavPoint{}, err
				}
				np.Title = title
			case "ol":
				children, err := parseNavList(dec, dir)
				if err != nil {
					return NavPoint{}, err
				}
				np.Children = children
			default:
				if err := dec.Skip(); err != nil {
					return NavPoint{}, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "li" {
				return np, nil
			}
		}
	}
}

// readText collects character data until it hits the closing tag for endLocal.
// It ignores nested elements.
func readText(dec *xml.Decoder, endLocal string) (string, error) {
	var buf strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("epub: read text: %w", err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			buf.Write(t)
		case xml.EndElement:
			if t.Name.Local == endLocal {
				return strings.TrimSpace(buf.String()), nil
			}
		}
	}
}
