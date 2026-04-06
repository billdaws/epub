package epub

import (
	"archive/zip"
	"fmt"
	"io"
)

// Reader holds an open EPUB file and its parsed Package, allowing content
// items to be read without reopening the archive on each call. The caller
// must call Close when done.
type Reader struct {
	// Package is the parsed OPF document for the open EPUB.
	Package *Package
	zr      *zip.ReadCloser
}

// Open opens the .epub file at path, parses its package document, and returns
// a Reader. The caller must call Close when done to release the file handle.
func Open(path string) (*Reader, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("epub: open: %w", err)
	}

	c, err := parseContainer(&zr.Reader)
	if err != nil {
		zr.Close()
		return nil, err
	}

	pkg, err := parsePackage(&zr.Reader, c.RootfilePath)
	if err != nil {
		zr.Close()
		return nil, err
	}

	return &Reader{Package: pkg, zr: zr}, nil
}

// Close closes the underlying EPUB file.
func (r *Reader) Close() error {
	return r.zr.Close()
}

// ReadItem returns the raw bytes of item's content file within the EPUB.
// item must come from r.Package.Manifest.
func (r *Reader) ReadItem(item Item) ([]byte, error) {
	f := findFile(&r.zr.Reader, item.Href)
	if f == nil {
		return nil, fmt.Errorf("epub: item %q not found at %q", item.ID, item.Href)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("epub: open item %q: %w", item.Href, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("epub: read item %q: %w", item.Href, err)
	}

	return data, nil
}
