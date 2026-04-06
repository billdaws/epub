package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"
)

const (
	writeOPFPath = "OEBPS/content.opf"
	writeOPFDir  = "OEBPS"
)

// Book is the input to Write: bibliographic metadata, content items, and the
// spine reading order.
type Book struct {
	Metadata Metadata      // bibliographic metadata written to the OPF <metadata> element
	Items    []ContentItem // content files added to the manifest and stored in the ZIP
	Spine    []string      // IDs of Items in reading order
}

// ContentItem is one content file to be written into the EPUB manifest and ZIP.
// Href is relative to the OPF document (e.g. "chapter1.xhtml"); Write stores
// it under OEBPS/ in the ZIP.
type ContentItem struct {
	ID         string // manifest item id attribute; must be unique within the Book
	Href       string // path relative to the OPF document, e.g. "chapter1.xhtml"
	MediaType  string // MIME type, e.g. "application/xhtml+xml"
	Properties string // space-separated EPUB 3 properties, e.g. "nav"
	Content    []byte // raw file bytes written into the ZIP
}

// Write encodes book as a valid EPUB 3 file and writes it to dst. It returns
// an error if any required metadata field (Title, Language, Identifier) is empty.
func Write(dst io.Writer, book Book) error {
	if err := checkBookMetadata(book.Metadata); err != nil {
		return err
	}

	zw := zip.NewWriter(dst)

	if err := writeMimetype(zw); err != nil {
		return err
	}
	if err := writeContainerXML(zw); err != nil {
		return err
	}
	if err := writeOPF(zw, book); err != nil {
		return err
	}
	for _, item := range book.Items {
		if err := writeContentItem(zw, item); err != nil {
			return err
		}
	}

	return zw.Close()
}

func checkBookMetadata(m Metadata) error {
	switch {
	case m.Title == "":
		return fmt.Errorf("epub: write: title is required")
	case m.Language == "":
		return fmt.Errorf("epub: write: language is required")
	case m.Identifier == "":
		return fmt.Errorf("epub: write: identifier is required")
	default:
		return nil
	}
}

// writeMimetype writes the EPUB mimetype entry. Per the EPUB spec it must be
// the first entry in the ZIP and must use STORE (no compression).
func writeMimetype(zw *zip.Writer) error {
	h := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	w, err := zw.CreateHeader(h)
	if err != nil {
		return fmt.Errorf("epub: create mimetype: %w", err)
	}
	_, err = io.WriteString(w, "application/epub+zip")
	return err
}

func writeContainerXML(zw *zip.Writer) error {
	w, err := zw.Create("META-INF/container.xml")
	if err != nil {
		return fmt.Errorf("epub: create container.xml: %w", err)
	}
	const body = `<?xml version="1.0" encoding="UTF-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	_, err = io.WriteString(w, body)
	return err
}

func writeOPF(zw *zip.Writer, book Book) error {
	w, err := zw.Create(writeOPFPath)
	if err != nil {
		return fmt.Errorf("epub: create OPF: %w", err)
	}
	return encodeOPF(w, book)
}

func writeContentItem(zw *zip.Writer, item ContentItem) error {
	w, err := zw.Create(writeOPFDir + "/" + item.Href)
	if err != nil {
		return fmt.Errorf("epub: create item %q: %w", item.Href, err)
	}
	_, err = w.Write(item.Content)
	return err
}

// encodeOPF writes the OPF package document for book to w.
func encodeOPF(w io.Writer, book Book) error {
	ew := &errWriter{w: w}

	esc := func(s string) string {
		var b strings.Builder
		xml.EscapeText(&b, []byte(s))
		return b.String()
	}

	ew.printf(`<?xml version="1.0" encoding="UTF-8"?>`)
	ew.printf("\n<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"3.0\" unique-identifier=\"uid\">")

	ew.printf("\n  <metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\">")
	ew.printf("\n    <dc:identifier id=\"uid\">%s</dc:identifier>", esc(book.Metadata.Identifier))
	ew.printf("\n    <dc:title>%s</dc:title>", esc(book.Metadata.Title))
	ew.printf("\n    <dc:language>%s</dc:language>", esc(book.Metadata.Language))
	for _, author := range book.Metadata.Authors {
		ew.printf("\n    <dc:creator>%s</dc:creator>", esc(author))
	}
	if book.Metadata.PublicationDate != "" {
		ew.printf("\n    <dc:date>%s</dc:date>", esc(book.Metadata.PublicationDate))
	}
	ew.printf("\n  </metadata>")

	ew.printf("\n  <manifest>")
	for _, item := range book.Items {
		ew.printf("\n    <item id=\"%s\" href=\"%s\" media-type=\"%s\"",
			esc(item.ID), esc(item.Href), esc(item.MediaType))
		if item.Properties != "" {
			ew.printf(" properties=\"%s\"", esc(item.Properties))
		}
		ew.printf("/>")
	}
	ew.printf("\n  </manifest>")

	ew.printf("\n  <spine>")
	for _, id := range book.Spine {
		ew.printf("\n    <itemref idref=\"%s\"/>", esc(id))
	}
	ew.printf("\n  </spine>")

	ew.printf("\n</package>\n")

	return ew.err
}

// errWriter wraps an io.Writer and records the first write error. It is safe
// for concurrent use.
type errWriter struct {
	mu  sync.Mutex
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	ew.mu.Lock()
	defer ew.mu.Unlock()
	if ew.err == nil {
		_, ew.err = fmt.Fprintf(ew.w, format, args...)
	}
}
