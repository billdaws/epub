package epub

import "fmt"

// FileNotFoundError is returned when a required file is absent from the EPUB
// ZIP archive. Path is the ZIP-relative path that was expected.
type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("epub: file not found at %q", e.Path)
}

// ItemNotFoundError is returned by [Reader.ReadItem] when the manifest item's
// content file is absent from the ZIP archive.
type ItemNotFoundError struct {
	ID   string // manifest item id attribute
	Href string // ZIP-relative path that was expected
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("epub: item %q not found at %q", e.ID, e.Href)
}

// MalformedContainerError is returned when META-INF/container.xml contains no
// usable rootfile entry.
type MalformedContainerError struct{}

func (e *MalformedContainerError) Error() string {
	return "epub: no rootfile found in container.xml"
}

// MissingTOCError is returned when neither an EPUB 3 navigation document nor
// an EPUB 2 NCX item is present in the OPF manifest.
type MissingTOCError struct{}

func (e *MissingTOCError) Error() string {
	return "epub: no table of contents found in manifest"
}

// MissingNavElementError is returned when the EPUB 3 navigation document
// contains no <nav epub:type="toc"> element.
type MissingNavElementError struct{}

func (e *MissingNavElementError) Error() string {
	return "epub: no toc nav element found in navigation document"
}

// MissingMetadataError is returned by [Write] when a required metadata field
// is empty. Field is the name of the missing field (e.g. "title").
type MissingMetadataError struct {
	Field string
}

func (e *MissingMetadataError) Error() string {
	return fmt.Sprintf("epub: write: %s is required", e.Field)
}
