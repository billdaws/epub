# epub — Go Library for EPUB Files

## Design Goals

1. **High test coverage.** Unit tests for all public APIs and integration tests using real public-domain books (e.g. from Project Gutenberg).
2. **Go idioms.** Unsurprising, conventional Go: clear error handling, no global state, errors as values, zero use of `panic` in library code.
3. **Zero dependencies.** Only the Go standard library. No third-party modules.

## Version-Dependent Logic

When adding parsing logic that differs between EPUB 2 and EPUB 3, follow this pattern:

- Read the version attribute early and error on unrecognised values — don't silently fall back.
- Dispatch to a version-specific private function (e.g. `extractMetadataV2` / `extractMetadataV3`) via a `switch` on the major version number.
- Factor shared logic into a common helper rather than duplicating it across version branches.
- Also expose public `DecodePackageV2` / `DecodePackageV3`-style entry points so callers who already know the version can bypass auto-detection.
- Keep version-agnostic concerns (manifest assembly, spine assembly, container parsing) outside the version switch entirely.

See `opf.go` for the reference implementation of this pattern.
