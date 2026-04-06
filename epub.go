// Package epub provides support for reading EPUB files.
package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
)

// Container represents a parsed EPUB container (META-INF/container.xml).
type Container struct {
	// RootfilePath is the path within the ZIP to the root OPF document,
	// e.g. "OEBPS/content.opf".
	RootfilePath string
}

// OpenContainer opens the .epub file at path and parses its container,
// returning the location of the root OPF document.
func OpenContainer(path string) (*Container, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("epub: open: %w", err)
	}
	defer zr.Close()
	return parseContainer(&zr.Reader)
}

func parseContainer(zr *zip.Reader) (*Container, error) {
	f := findFile(zr, "META-INF/container.xml")
	if f == nil {
		return nil, fmt.Errorf("epub: META-INF/container.xml not found")
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("epub: open container.xml: %w", err)
	}
	defer rc.Close()

	return decodeContainer(rc)
}

func decodeContainer(r io.Reader) (*Container, error) {
	var v struct {
		Rootfiles []struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.NewDecoder(r).Decode(&v); err != nil {
		return nil, fmt.Errorf("epub: decode container.xml: %w", err)
	}

	for _, rf := range v.Rootfiles {
		if rf.MediaType == "application/oebps-package+xml" && rf.FullPath != "" {
			return &Container{RootfilePath: rf.FullPath}, nil
		}
	}

	// Fall back to the first rootfile with a path if none matched the OPF media type.
	for _, rf := range v.Rootfiles {
		if rf.FullPath != "" {
			return &Container{RootfilePath: rf.FullPath}, nil
		}
	}

	return nil, fmt.Errorf("epub: no rootfile found in container.xml")
}

// findFile returns the named file from the ZIP, or nil if absent.
func findFile(zr *zip.Reader, name string) *zip.File {
	for _, f := range zr.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}
