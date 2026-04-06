package epub

import (
	"strings"
	"testing"
)

func TestDecodeContainer(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		wantPath string
		wantErr  bool
	}{
		{
			name: "standard OPF rootfile",
			xml: `<?xml version='1.0' encoding='utf-8'?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`,
			wantPath: "OEBPS/content.opf",
		},
		{
			name: "fallback to first rootfile when media-type differs",
			xml: `<?xml version='1.0' encoding='utf-8'?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="content.opf" media-type="application/xml"/>
  </rootfiles>
</container>`,
			wantPath: "content.opf",
		},
		{
			name: "OPF rootfile preferred over earlier non-OPF rootfile",
			xml: `<?xml version='1.0' encoding='utf-8'?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="other.xml" media-type="application/xml"/>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`,
			wantPath: "OEBPS/content.opf",
		},
		{
			name: "no rootfiles",
			xml: `<?xml version='1.0' encoding='utf-8'?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles/>
</container>`,
			wantErr: true,
		},
		{
			name:    "invalid XML",
			xml:     `not xml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := decodeContainer(strings.NewReader(tt.xml))
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeContainer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && c.RootfilePath != tt.wantPath {
				t.Errorf("RootfilePath = %q, want %q", c.RootfilePath, tt.wantPath)
			}
		})
	}
}

func TestOpenContainer_Integration(t *testing.T) {
	tests := []struct {
		file         string
		wantRootfile string
	}{
		{"testdata/gift-of-the-magi.epub", "OEBPS/content.opf"},
		{"testdata/importance-of-being-earnest.epub", "OEBPS/content.opf"},
		{"testdata/modest-proposal.epub", "OEBPS/content.opf"},
		{"testdata/wuthering-heights.epub", "OEBPS/content.opf"},
		{"testdata/yellow-wallpaper.epub", "OEBPS/content.opf"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			c, err := OpenContainer(tt.file)
			if err != nil {
				t.Fatalf("OpenContainer(%q): %v", tt.file, err)
			}
			if c.RootfilePath != tt.wantRootfile {
				t.Errorf("RootfilePath = %q, want %q", c.RootfilePath, tt.wantRootfile)
			}
		})
	}
}

func TestOpenContainer_Errors(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"missing file", "testdata/nonexistent.epub"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := OpenContainer(tt.path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
