package epub

import (
	"strings"
	"testing"
)

func TestDecodePackage(t *testing.T) {
	tests := []struct {
		name         string
		xml          string
		wantVersion  string
		wantTitle    string
		wantAuthors  []string
		wantLang     string
		wantID       string
		wantDate     string
		wantManifest []Item
		wantSpine    []SpineItem
		wantErr      bool
	}{
		{
			name: "EPUB 2 package",
			xml: `<?xml version='1.0' encoding='UTF-8'?>
<package xmlns:opf="http://www.idpf.org/2007/opf" xmlns:dc="http://purl.org/dc/elements/1.1/"
         xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="bookid">
  <metadata>
    <dc:identifier id="bookid" opf:scheme="URI">urn:isbn:12345</dc:identifier>
    <dc:title>My Book</dc:title>
    <dc:creator opf:file-as="Doe, Jane">Jane Doe</dc:creator>
    <dc:language>en</dc:language>
    <dc:date opf:event="publication">2001-01-01</dc:date>
    <dc:date opf:event="conversion">2026-01-01</dc:date>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.html" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx"  media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1" linear="yes"/>
  </spine>
</package>`,
			wantVersion: "2.0",
			wantTitle:   "My Book",
			wantAuthors: []string{"Jane Doe"},
			wantLang:    "en",
			wantID:      "urn:isbn:12345",
			wantDate:    "2001-01-01",
			wantManifest: []Item{
				{ID: "ch1", Href: "ch1.html", MediaType: "application/xhtml+xml"},
				{ID: "ncx", Href: "toc.ncx", MediaType: "application/x-dtbncx+xml"},
			},
			wantSpine: []SpineItem{
				{IDRef: "ch1", Linear: true},
			},
		},
		{
			name: "EPUB 3 package",
			xml: `<?xml version='1.0' encoding='UTF-8'?>
<package xmlns:dc="http://purl.org/dc/elements/1.1/"
         xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata>
    <dc:identifier id="uid">urn:uuid:abc</dc:identifier>
    <dc:title>Another Book</dc:title>
    <dc:creator id="a1">Alice Smith</dc:creator>
    <dc:creator id="a2">Bob Jones</dc:creator>
    <dc:language>fr</dc:language>
    <dc:date>2020-06-15</dc:date>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="nav" linear="no"/>
  </spine>
</package>`,
			wantVersion: "3.0",
			wantTitle:   "Another Book",
			wantAuthors: []string{"Alice Smith", "Bob Jones"},
			wantLang:    "fr",
			wantID:      "urn:uuid:abc",
			wantDate:    "2020-06-15",
			wantManifest: []Item{
				{ID: "nav", Href: "nav.xhtml", MediaType: "application/xhtml+xml", Properties: "nav"},
				{ID: "ch1", Href: "ch1.xhtml", MediaType: "application/xhtml+xml"},
			},
			wantSpine: []SpineItem{
				{IDRef: "ch1", Linear: true},
				{IDRef: "nav", Linear: false},
			},
		},
		{
			name:    "invalid XML",
			xml:     `not xml`,
			wantErr: true,
		},
		{
			name: "unknown version",
			xml: `<?xml version='1.0' encoding='UTF-8'?>
<package xmlns="http://www.idpf.org/2007/opf" xmlns:dc="http://purl.org/dc/elements/1.1/"
         version="4.0" unique-identifier="id">
  <metadata><dc:identifier id="id">x</dc:identifier><dc:title>T</dc:title><dc:language>en</dc:language></metadata>
  <manifest/><spine/>
</package>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := decodePackage(strings.NewReader(tt.xml), "content.opf")
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodePackage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if pkg.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", pkg.Version, tt.wantVersion)
			}
			if pkg.Metadata.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", pkg.Metadata.Title, tt.wantTitle)
			}
			if !stringSliceEqual(pkg.Metadata.Authors, tt.wantAuthors) {
				t.Errorf("Authors = %v, want %v", pkg.Metadata.Authors, tt.wantAuthors)
			}
			if pkg.Metadata.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", pkg.Metadata.Language, tt.wantLang)
			}
			if pkg.Metadata.Identifier != tt.wantID {
				t.Errorf("Identifier = %q, want %q", pkg.Metadata.Identifier, tt.wantID)
			}
			if pkg.Metadata.PublicationDate != tt.wantDate {
				t.Errorf("PublicationDate = %q, want %q", pkg.Metadata.PublicationDate, tt.wantDate)
			}
			if !itemSliceEqual(pkg.Manifest, tt.wantManifest) {
				t.Errorf("Manifest = %+v, want %+v", pkg.Manifest, tt.wantManifest)
			}
			if !spineSliceEqual(pkg.Spine, tt.wantSpine) {
				t.Errorf("Spine = %+v, want %+v", pkg.Spine, tt.wantSpine)
			}
		})
	}
}

func TestDecodePackageV2_SkipsVersionCheck(t *testing.T) {
	// DecodePackageV2 should succeed even when version attribute is missing or unknown,
	// because the caller has asserted the version externally.
	xml := `<?xml version='1.0' encoding='UTF-8'?>
<package xmlns="http://www.idpf.org/2007/opf" xmlns:dc="http://purl.org/dc/elements/1.1/"
         version="4.0" unique-identifier="id">
  <metadata>
    <dc:identifier id="id">x</dc:identifier>
    <dc:title>Force EPUB 2</dc:title>
    <dc:language>en</dc:language>
    <dc:date opf:event="publication" xmlns:opf="http://www.idpf.org/2007/opf">2000-01-01</dc:date>
  </metadata>
  <manifest/><spine/>
</package>`

	pkg, err := DecodePackageV2(strings.NewReader(xml), "content.opf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg.Metadata.Title != "Force EPUB 2" {
		t.Errorf("Title = %q, want %q", pkg.Metadata.Title, "Force EPUB 2")
	}
	if pkg.Metadata.PublicationDate != "2000-01-01" {
		t.Errorf("PublicationDate = %q, want %q", pkg.Metadata.PublicationDate, "2000-01-01")
	}
}

func TestDecodePackage_HrefPrefix(t *testing.T) {
	// When OPF is at OEBPS/content.opf, manifest hrefs should be prefixed with OEBPS/.
	xml := `<?xml version='1.0' encoding='UTF-8'?>
<package xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns="http://www.idpf.org/2007/opf"
         version="2.0" unique-identifier="id">
  <metadata>
    <dc:identifier id="id">x</dc:identifier>
    <dc:title>T</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.html" media-type="application/xhtml+xml"/>
  </manifest>
  <spine/>
</package>`

	pkg, err := decodePackage(strings.NewReader(xml), "OEBPS/content.opf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkg.Manifest) != 1 {
		t.Fatalf("want 1 manifest item, got %d", len(pkg.Manifest))
	}
	if pkg.Manifest[0].Href != "OEBPS/ch1.html" {
		t.Errorf("Href = %q, want %q", pkg.Manifest[0].Href, "OEBPS/ch1.html")
	}
}

func TestOpenPackage_Integration(t *testing.T) {
	tests := []struct {
		file        string
		wantTitle   string
		wantAuthors []string
		wantLang    string
		wantDate    string
		wantSpineN  int // expected number of spine items
	}{
		{
			file:        "testdata/gift-of-the-magi.epub",
			wantTitle:   "The Gift of the Magi",
			wantAuthors: []string{"O. Henry"},
			wantLang:    "en",
			wantDate:    "2005-01-01",
			wantSpineN:  4,
		},
		{
			file:        "testdata/importance-of-being-earnest.epub",
			wantTitle:   "The Importance of Being Earnest: A Trivial Comedy for Serious People",
			wantAuthors: []string{"Oscar Wilde"},
			wantLang:    "en",
			wantDate:    "1997-03-01",
			wantSpineN:  6,
		},
		{
			file:        "testdata/wuthering-heights.epub",
			wantTitle:   "Wuthering Heights",
			wantAuthors: []string{"Emily Brontë"},
			wantLang:    "en",
			wantDate:    "1996-12-01",
			wantSpineN:  37,
		},
		{
			file:        "testdata/yellow-wallpaper.epub",
			wantTitle:   "The Yellow Wallpaper",
			wantAuthors: []string{"Charlotte Perkins Gilman"},
			wantLang:    "en",
			wantDate:    "1999-11-01",
			wantSpineN:  3,
		},
		{
			file:        "testdata/modest-proposal.epub",
			wantTitle:   "A Modest Proposal / For preventing the children of poor people in Ireland, from being a burden on their parents or country, and for making them beneficial to the publick",
			wantAuthors: []string{"Jonathan Swift"},
			wantLang:    "en",
			wantDate:    "1997-10-01",
			wantSpineN:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			pkg, err := OpenPackage(tt.file)
			if err != nil {
				t.Fatalf("OpenPackage(%q): %v", tt.file, err)
			}
			if pkg.Metadata.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", pkg.Metadata.Title, tt.wantTitle)
			}
			if !stringSliceEqual(pkg.Metadata.Authors, tt.wantAuthors) {
				t.Errorf("Authors = %v, want %v", pkg.Metadata.Authors, tt.wantAuthors)
			}
			if pkg.Metadata.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", pkg.Metadata.Language, tt.wantLang)
			}
			if pkg.Metadata.PublicationDate != tt.wantDate {
				t.Errorf("PublicationDate = %q, want %q", pkg.Metadata.PublicationDate, tt.wantDate)
			}
			if len(pkg.Spine) != tt.wantSpineN {
				t.Errorf("len(Spine) = %d, want %d", len(pkg.Spine), tt.wantSpineN)
			}
			if len(pkg.Manifest) == 0 {
				t.Error("Manifest is empty")
			}
		})
	}
}

// helpers

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func itemSliceEqual(a, b []Item) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func spineSliceEqual(a, b []SpineItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
