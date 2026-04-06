package epub

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// testBook returns a minimal valid Book for use across write tests.
func testBook() Book {
	return Book{
		Metadata: Metadata{
			Title:           "A Test Book",
			Authors:         []string{"Alice Example", "Bob Example"},
			Language:        "en",
			Identifier:      "urn:uuid:test-write-1234",
			PublicationDate: "2024-01-01",
		},
		Items: []ContentItem{
			{
				ID:         "nav",
				Href:       "nav.xhtml",
				MediaType:  "application/xhtml+xml",
				Properties: "nav",
				Content: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><title>Table of Contents</title></head>
<body>
  <nav epub:type="toc"><ol><li><a href="chapter1.xhtml">Chapter 1</a></li></ol></nav>
</body>
</html>`),
			},
			{
				ID:        "c1",
				Href:      "chapter1.xhtml",
				MediaType: "application/xhtml+xml",
				Content:   []byte(`<html><body><p>Hello, world.</p></body></html>`),
			},
		},
		Spine: []string{"c1"},
	}
}

// writeTempEPUB writes book to a temporary file and returns its path.
// The file is removed when the test ends.
func writeTempEPUB(t *testing.T, book Book) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.epub")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer f.Close()
	if err := Write(f, book); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return path
}

func TestWrite_RoundTrip(t *testing.T) {
	book := testBook()
	path := writeTempEPUB(t, book)

	pkg, err := OpenPackage(path)
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}

	// Metadata.
	if pkg.Metadata.Title != book.Metadata.Title {
		t.Errorf("Title = %q, want %q", pkg.Metadata.Title, book.Metadata.Title)
	}
	if pkg.Metadata.Language != book.Metadata.Language {
		t.Errorf("Language = %q, want %q", pkg.Metadata.Language, book.Metadata.Language)
	}
	if pkg.Metadata.Identifier != book.Metadata.Identifier {
		t.Errorf("Identifier = %q, want %q", pkg.Metadata.Identifier, book.Metadata.Identifier)
	}
	if pkg.Metadata.PublicationDate != book.Metadata.PublicationDate {
		t.Errorf("PublicationDate = %q, want %q", pkg.Metadata.PublicationDate, book.Metadata.PublicationDate)
	}
	if len(pkg.Metadata.Authors) != len(book.Metadata.Authors) {
		t.Errorf("Authors = %v, want %v", pkg.Metadata.Authors, book.Metadata.Authors)
	}

	// Manifest: Href round-trips with the OEBPS/ prefix added by the parser.
	if len(pkg.Manifest) != len(book.Items) {
		t.Fatalf("len(Manifest) = %d, want %d", len(pkg.Manifest), len(book.Items))
	}
	for i, item := range pkg.Manifest {
		want := book.Items[i]
		if item.ID != want.ID {
			t.Errorf("Manifest[%d].ID = %q, want %q", i, item.ID, want.ID)
		}
		wantHref := writeOPFDir + "/" + want.Href
		if item.Href != wantHref {
			t.Errorf("Manifest[%d].Href = %q, want %q", i, item.Href, wantHref)
		}
		if item.MediaType != want.MediaType {
			t.Errorf("Manifest[%d].MediaType = %q, want %q", i, item.MediaType, want.MediaType)
		}
		if item.Properties != want.Properties {
			t.Errorf("Manifest[%d].Properties = %q, want %q", i, item.Properties, want.Properties)
		}
	}

	// Spine.
	if len(pkg.Spine) != len(book.Spine) {
		t.Fatalf("len(Spine) = %d, want %d", len(pkg.Spine), len(book.Spine))
	}
	for i, si := range pkg.Spine {
		if si.IDRef != book.Spine[i] {
			t.Errorf("Spine[%d].IDRef = %q, want %q", i, si.IDRef, book.Spine[i])
		}
	}
}

func TestWrite_Validate(t *testing.T) {
	path := writeTempEPUB(t, testBook())

	pkg, err := OpenPackage(path)
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}

	if vs := Validate(pkg); len(vs) != 0 {
		t.Errorf("Validate: unexpected violations: %v", vs)
	}
}

func TestWrite_ContentRoundTrip(t *testing.T) {
	book := testBook()
	path := writeTempEPUB(t, book)

	r, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()

	byID := make(map[string]Item, len(r.Package.Manifest))
	for _, item := range r.Package.Manifest {
		byID[item.ID] = item
	}

	for _, ci := range book.Items {
		item, ok := byID[ci.ID]
		if !ok {
			t.Errorf("item %q not found in manifest after round-trip", ci.ID)
			continue
		}
		got, err := r.ReadItem(item)
		if err != nil {
			t.Errorf("ReadItem(%q): %v", ci.ID, err)
			continue
		}
		if !bytes.Equal(got, ci.Content) {
			t.Errorf("ReadItem(%q): content mismatch\ngot:  %q\nwant: %q", ci.ID, got, ci.Content)
		}
	}
}

func TestErrWriter_ConcurrentPrintf(t *testing.T) {
	// Verify that concurrent calls to printf do not race. Run with -race.
	ew := &errWriter{w: io.Discard}
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			ew.printf("hello %s", "world")
		}()
	}
	wg.Wait()
}

// TestWrite_ConcurrentWriteAndRead verifies that goroutines writing new EPUBs
// and goroutines reading an existing EPUB can run simultaneously without data
// races. Run with -race.
func TestWrite_ConcurrentWriteAndRead(t *testing.T) {
	book := testBook()
	// Write once up front so the readers have a valid file to open.
	readPath := writeTempEPUB(t, book)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// One set of goroutines writes new EPUBs to independent buffers.
	for range goroutines {
		go func() {
			defer wg.Done()
			if err := Write(&bytes.Buffer{}, book); err != nil {
				t.Errorf("Write: %v", err)
			}
		}()
	}

	// Another set opens and reads the same file concurrently with the writes.
	for range goroutines {
		go func() {
			defer wg.Done()
			r, err := Open(readPath)
			if err != nil {
				t.Errorf("Open: %v", err)
				return
			}
			defer r.Close()
			for _, item := range r.Package.Manifest {
				if _, err := r.ReadItem(item); err != nil {
					t.Errorf("ReadItem(%q): %v", item.ID, err)
				}
			}
		}()
	}

	wg.Wait()
}

func TestWrite_Errors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Book)
	}{
		{"missing title", func(b *Book) { b.Metadata.Title = "" }},
		{"missing language", func(b *Book) { b.Metadata.Language = "" }},
		{"missing identifier", func(b *Book) { b.Metadata.Identifier = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := testBook()
			tt.mutate(&book)
			err := Write(&bytes.Buffer{}, book)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
