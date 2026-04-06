package epub

import (
	"fmt"
	"strings"
)

// ViolationCode identifies the structural rule that was violated.
type ViolationCode string

const (
	// Metadata violations.
	ViolationMissingTitle      ViolationCode = "missing-title"
	ViolationMissingLanguage   ViolationCode = "missing-language"
	ViolationMissingIdentifier ViolationCode = "missing-identifier"

	// Manifest violations.
	ViolationEmptyManifest        ViolationCode = "empty-manifest"
	ViolationMissingItemID        ViolationCode = "missing-item-id"
	ViolationMissingItemHref      ViolationCode = "missing-item-href"
	ViolationMissingItemMediaType ViolationCode = "missing-item-media-type"
	ViolationDuplicateItemID      ViolationCode = "duplicate-item-id"
	ViolationDuplicateItemHref    ViolationCode = "duplicate-item-href"

	// Spine violations.
	ViolationEmptySpine       ViolationCode = "empty-spine"
	ViolationBrokenSpineIDRef ViolationCode = "broken-spine-idref"

	// Version-specific structural violations.
	ViolationMissingNav ViolationCode = "missing-nav" // EPUB 3: nav document required
	ViolationMissingNCX ViolationCode = "missing-ncx" // EPUB 2: NCX document required
)

// Violation describes a single structural rule violation in a Package.
type Violation struct {
	Code    ViolationCode
	Message string
}

// violations accumulates Violation values.
type violations []Violation

func (vs *violations) add(code ViolationCode, format string, args ...any) {
	*vs = append(*vs, Violation{Code: code, Message: fmt.Sprintf(format, args...)})
}

// Validate checks that pkg conforms to the structural rules for its EPUB
// version and returns all violations found. An empty slice means the package
// is valid. Validate does not open any files; it inspects only the parsed
// Package value.
func Validate(pkg *Package) []Violation {
	var vs violations
	vs.validateMetadata(pkg.Metadata)
	seenID := vs.validateManifest(pkg.Manifest)
	vs.validateSpine(pkg.Spine, seenID)
	vs.validateVersion(pkg.Version, pkg.Manifest)
	return []Violation(vs)
}

func (vs *violations) validateMetadata(m Metadata) {
	if m.Title == "" {
		vs.add(ViolationMissingTitle, "dc:title is required")
	}
	if m.Language == "" {
		vs.add(ViolationMissingLanguage, "dc:language is required")
	}
	if m.Identifier == "" {
		vs.add(ViolationMissingIdentifier, "dc:identifier is required")
	}
}

// validateManifest checks manifest items and returns the set of valid IDs seen.
func (vs *violations) validateManifest(manifest []Item) map[string]bool {
	seenID := make(map[string]bool, len(manifest))
	seenHref := make(map[string]bool, len(manifest))

	if len(manifest) == 0 {
		vs.add(ViolationEmptyManifest, "manifest must contain at least one item")
		return seenID
	}

	for _, item := range manifest {
		switch {
		case item.ID == "":
			vs.add(ViolationMissingItemID, "manifest item with href %q has no id", item.Href)
		case seenID[item.ID]:
			vs.add(ViolationDuplicateItemID, "manifest item id %q appears more than once", item.ID)
		default:
			seenID[item.ID] = true
		}

		switch {
		case item.Href == "":
			vs.add(ViolationMissingItemHref, "manifest item %q has no href", item.ID)
		case seenHref[item.Href]:
			vs.add(ViolationDuplicateItemHref, "manifest item href %q appears more than once", item.Href)
		default:
			seenHref[item.Href] = true
		}

		if item.MediaType == "" {
			vs.add(ViolationMissingItemMediaType, "manifest item %q has no media-type", item.ID)
		}
	}

	return seenID
}

func (vs *violations) validateSpine(spine []SpineItem, seenID map[string]bool) {
	if len(spine) == 0 {
		vs.add(ViolationEmptySpine, "spine must contain at least one itemref")
		return
	}
	for _, si := range spine {
		if !seenID[si.IDRef] {
			vs.add(ViolationBrokenSpineIDRef, "spine itemref %q does not match any manifest item", si.IDRef)
		}
	}
}

func (vs *violations) validateVersion(version string, manifest []Item) {
	majorVersion, _, _ := strings.Cut(version, ".")
	switch majorVersion {
	case "3":
		if !hasNav(manifest) {
			vs.add(ViolationMissingNav, "EPUB 3 package must include a nav document (manifest item with media-type %q and properties containing \"nav\")", "application/xhtml+xml")
		}
	case "2":
		if !hasNCX(manifest) {
			vs.add(ViolationMissingNCX, "EPUB 2 package must include an NCX document (manifest item with media-type %q)", "application/x-dtbncx+xml")
		}
	}
}

// hasNav reports whether the manifest contains an EPUB 3 navigation document.
func hasNav(manifest []Item) bool {
	for _, item := range manifest {
		if item.MediaType == "application/xhtml+xml" && containsWord(item.Properties, "nav") {
			return true
		}
	}
	return false
}

// hasNCX reports whether the manifest contains an EPUB 2 NCX document.
func hasNCX(manifest []Item) bool {
	for _, item := range manifest {
		if item.MediaType == "application/x-dtbncx+xml" {
			return true
		}
	}
	return false
}
