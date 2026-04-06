package epub

import (
	"strings"
	"testing"
)

func TestDecodeNCX(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		dir     string
		want    []NavPoint
		wantErr bool
	}{
		{
			name: "flat navMap",
			dir:  "OEBPS",
			xml: `<?xml version='1.0' encoding='UTF-8'?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np-1" playOrder="1">
      <navLabel><text>Chapter One</text></navLabel>
      <content src="ch1.html#start"/>
    </navPoint>
    <navPoint id="np-2" playOrder="2">
      <navLabel><text>Chapter Two</text></navLabel>
      <content src="ch2.html"/>
    </navPoint>
  </navMap>
</ncx>`,
			want: []NavPoint{
				{Title: "Chapter One", Src: "OEBPS/ch1.html#start"},
				{Title: "Chapter Two", Src: "OEBPS/ch2.html"},
			},
		},
		{
			name: "nested navPoints",
			dir:  "OEBPS",
			xml: `<?xml version='1.0' encoding='UTF-8'?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np-1" playOrder="1">
      <navLabel><text>Part One</text></navLabel>
      <content src="part1.html"/>
      <navPoint id="np-2" playOrder="2">
        <navLabel><text>Chapter One</text></navLabel>
        <content src="ch1.html"/>
      </navPoint>
    </navPoint>
  </navMap>
</ncx>`,
			want: []NavPoint{
				{
					Title: "Part One",
					Src:   "OEBPS/part1.html",
					Children: []NavPoint{
						{Title: "Chapter One", Src: "OEBPS/ch1.html"},
					},
				},
			},
		},
		{
			name: "whitespace in label is trimmed",
			dir:  ".",
			xml: `<?xml version='1.0' encoding='UTF-8'?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np-1" playOrder="1">
      <navLabel><text>  Trimmed Title  </text></navLabel>
      <content src="ch1.html"/>
    </navPoint>
  </navMap>
</ncx>`,
			want: []NavPoint{
				{Title: "Trimmed Title", Src: "ch1.html"},
			},
		},
		{
			name:    "invalid XML",
			xml:     `not xml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeNCX(strings.NewReader(tt.xml), tt.dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeNCX() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !navPointSliceEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestDecodeNav(t *testing.T) {
	tests := []struct {
		name    string
		xhtml   string
		dir     string
		want    []NavPoint
		wantErr bool
	}{
		{
			name: "flat toc nav",
			dir:  "OEBPS",
			xhtml: `<?xml version='1.0' encoding='UTF-8'?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <body>
    <nav epub:type="toc">
      <ol>
        <li><a href="ch1.xhtml#s1">Chapter One</a></li>
        <li><a href="ch2.xhtml">Chapter Two</a></li>
      </ol>
    </nav>
  </body>
</html>`,
			want: []NavPoint{
				{Title: "Chapter One", Src: "OEBPS/ch1.xhtml#s1"},
				{Title: "Chapter Two", Src: "OEBPS/ch2.xhtml"},
			},
		},
		{
			name: "nested ol",
			dir:  "OEBPS",
			xhtml: `<?xml version='1.0' encoding='UTF-8'?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <body>
    <nav epub:type="toc">
      <ol>
        <li>
          <a href="part1.xhtml">Part One</a>
          <ol>
            <li><a href="ch1.xhtml">Chapter One</a></li>
          </ol>
        </li>
      </ol>
    </nav>
  </body>
</html>`,
			want: []NavPoint{
				{
					Title: "Part One",
					Src:   "OEBPS/part1.xhtml",
					Children: []NavPoint{
						{Title: "Chapter One", Src: "OEBPS/ch1.xhtml"},
					},
				},
			},
		},
		{
			name: "non-toc nav is skipped",
			dir:  "OEBPS",
			xhtml: `<?xml version='1.0' encoding='UTF-8'?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <body>
    <nav epub:type="landmarks">
      <ol><li><a href="ch1.xhtml">Start</a></li></ol>
    </nav>
    <nav epub:type="toc">
      <ol><li><a href="ch1.xhtml">Chapter One</a></li></ol>
    </nav>
  </body>
</html>`,
			want: []NavPoint{
				{Title: "Chapter One", Src: "OEBPS/ch1.xhtml"},
			},
		},
		{
			name: "no toc nav returns error",
			xhtml: `<?xml version='1.0' encoding='UTF-8'?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <body><nav epub:type="landmarks"><ol/></nav></body>
</html>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeNav(strings.NewReader(tt.xhtml), tt.dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeNav() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !navPointSliceEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestOpenTOC_Integration(t *testing.T) {
	tests := []struct {
		file           string
		wantFirstTitle string
		wantLen        int // top-level entries
	}{
		{
			file:           "testdata/gift-of-the-magi.epub",
			wantFirstTitle: "The Gift of the Magi",
			wantLen:        2,
		},
		{
			file:           "testdata/importance-of-being-earnest.epub",
			wantFirstTitle: "The Importance of Being Earnest",
			wantLen:        9,
		},
		{
			file:           "testdata/modest-proposal.epub",
			wantFirstTitle: "A Modest Proposal",
			wantLen:        4,
		},
		{
			file:           "testdata/wuthering-heights.epub",
			wantFirstTitle: "Wuthering Heights",
			wantLen:        36,
		},
		{
			file:           "testdata/yellow-wallpaper.epub",
			wantFirstTitle: "The Yellow Wallpaper",
			wantLen:        2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			toc, err := OpenTOC(tt.file)
			if err != nil {
				t.Fatalf("OpenTOC(%q): %v", tt.file, err)
			}
			if len(toc) != tt.wantLen {
				t.Errorf("len(toc) = %d, want %d", len(toc), tt.wantLen)
			}
			if len(toc) > 0 && toc[0].Title != tt.wantFirstTitle {
				t.Errorf("toc[0].Title = %q, want %q", toc[0].Title, tt.wantFirstTitle)
			}
			for i, np := range toc {
				if np.Src == "" {
					t.Errorf("toc[%d].Src is empty", i)
				}
			}
		})
	}
}

// helpers

func navPointSliceEqual(a, b []NavPoint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Src != b[i].Src {
			return false
		}
		if !navPointSliceEqual(a[i].Children, b[i].Children) {
			return false
		}
	}
	return true
}
