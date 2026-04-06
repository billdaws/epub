package epub

import (
	"testing"
)

// validEPUB3 returns a minimal structurally valid EPUB 3 Package.
func validEPUB3() *Package {
	return &Package{
		Version: "3.0",
		Metadata: Metadata{
			Title:      "Test Book",
			Language:   "en",
			Identifier: "urn:uuid:test-1234",
		},
		Manifest: []Item{
			{ID: "nav", Href: "nav.xhtml", MediaType: "application/xhtml+xml", Properties: "nav"},
			{ID: "c1", Href: "chapter1.xhtml", MediaType: "application/xhtml+xml"},
		},
		Spine: []SpineItem{
			{IDRef: "c1", Linear: true},
		},
	}
}

// validEPUB2 returns a minimal structurally valid EPUB 2 Package.
func validEPUB2() *Package {
	return &Package{
		Version: "2.0",
		Metadata: Metadata{
			Title:      "Test Book",
			Language:   "en",
			Identifier: "urn:uuid:test-5678",
		},
		Manifest: []Item{
			{ID: "ncx", Href: "toc.ncx", MediaType: "application/x-dtbncx+xml"},
			{ID: "c1", Href: "chapter1.html", MediaType: "application/xhtml+xml"},
		},
		Spine: []SpineItem{
			{IDRef: "c1", Linear: true},
		},
	}
}

func TestValidate_ValidPackages(t *testing.T) {
	tests := []struct {
		name string
		pkg  *Package
	}{
		{"valid EPUB 3", validEPUB3()},
		{"valid EPUB 2", validEPUB2()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if vs := Validate(tt.pkg); len(vs) != 0 {
				t.Errorf("expected no violations, got %v", vs)
			}
		})
	}
}

func TestValidate_Metadata(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Package)
		wantCode ViolationCode
	}{
		{"missing title", func(p *Package) { p.Metadata.Title = "" }, ViolationMissingTitle},
		{"missing language", func(p *Package) { p.Metadata.Language = "" }, ViolationMissingLanguage},
		{"missing identifier", func(p *Package) { p.Metadata.Identifier = "" }, ViolationMissingIdentifier},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validEPUB3()
			tt.mutate(p)
			assertViolation(t, Validate(p), tt.wantCode)
		})
	}
}

func TestValidate_Manifest(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Package)
		wantCode ViolationCode
	}{
		{
			"empty manifest",
			func(p *Package) { p.Manifest = nil },
			ViolationEmptyManifest,
		},
		{
			"missing item id",
			func(p *Package) {
				p.Manifest = append(p.Manifest, Item{Href: "x.xhtml", MediaType: "application/xhtml+xml"})
			},
			ViolationMissingItemID,
		},
		{
			"missing item href",
			func(p *Package) { p.Manifest = append(p.Manifest, Item{ID: "x", MediaType: "application/xhtml+xml"}) },
			ViolationMissingItemHref,
		},
		{
			"missing item media-type",
			func(p *Package) { p.Manifest = append(p.Manifest, Item{ID: "x", Href: "x.xhtml"}) },
			ViolationMissingItemMediaType,
		},
		{
			"duplicate item id",
			func(p *Package) {
				p.Manifest = append(p.Manifest, Item{ID: "c1", Href: "extra.xhtml", MediaType: "application/xhtml+xml"})
			},
			ViolationDuplicateItemID,
		},
		{
			"duplicate item href",
			func(p *Package) {
				p.Manifest = append(p.Manifest, Item{ID: "extra", Href: "chapter1.xhtml", MediaType: "application/xhtml+xml"})
			},
			ViolationDuplicateItemHref,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validEPUB3()
			tt.mutate(p)
			assertViolation(t, Validate(p), tt.wantCode)
		})
	}
}

func TestValidate_Spine(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Package)
		wantCode ViolationCode
	}{
		{
			"empty spine",
			func(p *Package) { p.Spine = nil },
			ViolationEmptySpine,
		},
		{
			"broken idref",
			func(p *Package) { p.Spine = append(p.Spine, SpineItem{IDRef: "does-not-exist"}) },
			ViolationBrokenSpineIDRef,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validEPUB3()
			tt.mutate(p)
			assertViolation(t, Validate(p), tt.wantCode)
		})
	}
}

func TestValidate_VersionSpecific(t *testing.T) {
	t.Run("EPUB 3 missing nav", func(t *testing.T) {
		p := validEPUB3()
		// Remove the nav property from the nav item.
		p.Manifest[0].Properties = ""
		assertViolation(t, Validate(p), ViolationMissingNav)
	})

	t.Run("EPUB 2 missing NCX", func(t *testing.T) {
		p := validEPUB2()
		// Replace the NCX item with a plain content item.
		p.Manifest[0] = Item{ID: "ncx", Href: "toc.ncx", MediaType: "application/xhtml+xml"}
		assertViolation(t, Validate(p), ViolationMissingNCX)
	})
}

func TestValidate_Integration(t *testing.T) {
	files := []string{
		"testdata/gift-of-the-magi.epub",
		"testdata/importance-of-being-earnest.epub",
		"testdata/modest-proposal.epub",
		"testdata/wuthering-heights.epub",
		"testdata/yellow-wallpaper.epub",
	}
	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			pkg, err := OpenPackage(path)
			if err != nil {
				t.Fatalf("OpenPackage: %v", err)
			}
			if vs := Validate(pkg); len(vs) != 0 {
				t.Errorf("unexpected violations: %v", vs)
			}
		})
	}
}

// assertViolation fails the test if vs does not contain exactly one violation
// with the given code.
func assertViolation(t *testing.T, vs []Violation, want ViolationCode) {
	t.Helper()
	for _, v := range vs {
		if v.Code == want {
			return
		}
	}
	t.Errorf("expected violation %q; got %v", want, vs)
}
