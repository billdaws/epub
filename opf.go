package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

// Package is the parsed contents of an OPF package document.
type Package struct {
	Version  string // e.g. "2.0" or "3.0", as written in the OPF
	Metadata Metadata
	Manifest []Item
	Spine    []SpineItem
}

// Metadata holds Dublin Core bibliographic information from the OPF.
type Metadata struct {
	Title           string
	Authors         []string
	Language        string
	Identifier      string
	PublicationDate string // "YYYY-MM-DD" or "YYYY" as written in the file; empty if absent
}

// Item is an entry in the OPF manifest.
type Item struct {
	ID        string
	Href      string // relative to the OPF document's directory
	MediaType string
	// Properties contains space-separated property values (EPUB 3 only, e.g. "nav", "cover-image").
	// Empty for EPUB 2 items.
	Properties string
}

// SpineItem is one entry in the OPF spine, identifying a manifest item by ID.
type SpineItem struct {
	IDRef  string
	Linear bool // false only when the OPF explicitly sets linear="no"
}

// OpenPackage opens the .epub file at path, locates the OPF document via the
// container, and returns the parsed Package.
func OpenPackage(path string) (*Package, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("epub: open: %w", err)
	}
	defer zr.Close()

	c, err := parseContainer(&zr.Reader)
	if err != nil {
		return nil, err
	}

	return parsePackage(&zr.Reader, c.RootfilePath)
}

func parsePackage(zr *zip.Reader, opfPath string) (*Package, error) {
	f := findFile(zr, opfPath)
	if f == nil {
		return nil, fmt.Errorf("epub: OPF not found at %q", opfPath)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("epub: open OPF: %w", err)
	}
	defer rc.Close()

	return decodePackage(rc, opfPath)
}

// xmlPackage mirrors the OPF XML structure for decoding.
type xmlPackage struct {
	XMLName          xml.Name `xml:"http://www.idpf.org/2007/opf package"`
	Version          string   `xml:"version,attr"`
	UniqueIdentifier string   `xml:"unique-identifier,attr"`
	Metadata         struct {
		Titles   []string `xml:"http://purl.org/dc/elements/1.1/ title"`
		Creators []struct {
			Value string `xml:",chardata"`
		} `xml:"http://purl.org/dc/elements/1.1/ creator"`
		Languages   []string `xml:"http://purl.org/dc/elements/1.1/ language"`
		Identifiers []struct {
			Value string `xml:",chardata"`
			ID    string `xml:"id,attr"`
		} `xml:"http://purl.org/dc/elements/1.1/ identifier"`
		Dates []struct {
			Value string `xml:",chardata"`
			Event string `xml:"http://www.idpf.org/2007/opf event,attr"`
		} `xml:"http://purl.org/dc/elements/1.1/ date"`
	} `xml:"http://www.idpf.org/2007/opf metadata"`
	Manifest struct {
		Items []struct {
			ID         string `xml:"id,attr"`
			Href       string `xml:"href,attr"`
			MediaType  string `xml:"media-type,attr"`
			Properties string `xml:"properties,attr"`
		} `xml:"http://www.idpf.org/2007/opf item"`
	} `xml:"http://www.idpf.org/2007/opf manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef  string `xml:"idref,attr"`
			Linear string `xml:"linear,attr"`
		} `xml:"http://www.idpf.org/2007/opf itemref"`
	} `xml:"http://www.idpf.org/2007/opf spine"`
}

func parseXMLPackage(r io.Reader, opfPath string) (xmlPackage, error) {
	var x xmlPackage
	if err := xml.NewDecoder(r).Decode(&x); err != nil {
		return xmlPackage{}, fmt.Errorf("epub: decode OPF %q: %w", opfPath, err)
	}
	return x, nil
}

// DecodePackageV2 parses r as an EPUB 2 OPF document. It ignores the version
// attribute; use this when you already know the content is EPUB 2.
func DecodePackageV2(r io.Reader, opfPath string) (*Package, error) {
	x, err := parseXMLPackage(r, opfPath)
	if err != nil {
		return nil, err
	}
	return buildPackage(x, extractMetadataV2(x), opfPath), nil
}

// DecodePackageV3 parses r as an EPUB 3 OPF document. It ignores the version
// attribute; use this when you already know the content is EPUB 3.
func DecodePackageV3(r io.Reader, opfPath string) (*Package, error) {
	x, err := parseXMLPackage(r, opfPath)
	if err != nil {
		return nil, err
	}
	return buildPackage(x, extractMetadataV3(x), opfPath), nil
}

// decodePackage reads the version attribute and dispatches to the appropriate
// version-specific decoder. It returns an error for unrecognised versions.
func decodePackage(r io.Reader, opfPath string) (*Package, error) {
	x, err := parseXMLPackage(r, opfPath)
	if err != nil {
		return nil, err
	}

	majorVersion, _, _ := strings.Cut(x.Version, ".")
	switch majorVersion {
	case "2":
		return buildPackage(x, extractMetadataV2(x), opfPath), nil
	case "3":
		return buildPackage(x, extractMetadataV3(x), opfPath), nil
	default:
		return nil, fmt.Errorf("epub: unsupported OPF version %q in %q", x.Version, opfPath)
	}
}

// buildPackage assembles a Package from a decoded xmlPackage and pre-extracted
// metadata. Manifest and spine parsing is identical across EPUB versions.
func buildPackage(x xmlPackage, meta Metadata, opfPath string) *Package {
	pkg := &Package{
		Version:  x.Version,
		Metadata: meta,
	}

	opfDir := path.Dir(opfPath)
	pkg.Manifest = make([]Item, 0, len(x.Manifest.Items))
	for _, xi := range x.Manifest.Items {
		href := xi.Href
		if opfDir != "." {
			href = opfDir + "/" + xi.Href
		}
		pkg.Manifest = append(pkg.Manifest, Item{
			ID:         xi.ID,
			Href:       href,
			MediaType:  xi.MediaType,
			Properties: xi.Properties,
		})
	}

	pkg.Spine = make([]SpineItem, 0, len(x.Spine.ItemRefs))
	for _, xi := range x.Spine.ItemRefs {
		pkg.Spine = append(pkg.Spine, SpineItem{
			IDRef:  xi.IDRef,
			Linear: xi.Linear != "no",
		})
	}

	return pkg
}

// extractMetadataV2 extracts metadata from an EPUB 2 OPF. It prefers the
// dc:date element with opf:event="publication" for the publication date.
func extractMetadataV2(x xmlPackage) Metadata {
	m := extractCommonMetadata(x)

	for _, d := range x.Metadata.Dates {
		if d.Event == "publication" {
			m.PublicationDate = strings.TrimSpace(d.Value)
			break
		}
	}
	if m.PublicationDate == "" && len(x.Metadata.Dates) > 0 {
		m.PublicationDate = strings.TrimSpace(x.Metadata.Dates[0].Value)
	}

	return m
}

// extractMetadataV3 extracts metadata from an EPUB 3 OPF. It uses the first
// dc:date element with no opf:event attribute for the publication date.
func extractMetadataV3(x xmlPackage) Metadata {
	m := extractCommonMetadata(x)

	for _, d := range x.Metadata.Dates {
		if d.Event == "" {
			m.PublicationDate = strings.TrimSpace(d.Value)
			break
		}
	}

	return m
}

// extractCommonMetadata handles the metadata fields that are identical across
// all EPUB versions: title, authors, language, and identifier.
func extractCommonMetadata(x xmlPackage) Metadata {
	m := Metadata{}

	if len(x.Metadata.Titles) > 0 {
		m.Title = x.Metadata.Titles[0]
	}

	for _, c := range x.Metadata.Creators {
		if v := strings.TrimSpace(c.Value); v != "" {
			m.Authors = append(m.Authors, v)
		}
	}

	if len(x.Metadata.Languages) > 0 {
		m.Language = x.Metadata.Languages[0]
	}

	// Use the identifier whose XML id matches unique-identifier; fall back to first.
	for _, id := range x.Metadata.Identifiers {
		if id.ID == x.UniqueIdentifier {
			m.Identifier = strings.TrimSpace(id.Value)
			break
		}
	}
	if m.Identifier == "" && len(x.Metadata.Identifiers) > 0 {
		m.Identifier = strings.TrimSpace(x.Metadata.Identifiers[0].Value)
	}

	return m
}
