# Contributing

## Design goals

**High test coverage.** Every public API should have unit tests. Where possible, use real public-domain EPUB files (e.g. from Project Gutenberg) as integration fixtures rather than synthetic ones — they catch real-world edge cases that hand-crafted test data tends to miss.

**Go idioms.** Aim for conventional, unsurprising Go. No panics. No global state. Error handling should be explicit and local. If something feels like it needs a clever abstraction, it probably just needs a clearer function name.

**Zero dependencies.** Only the Go standard library. Do not add third-party modules.

## Adding EPUB version-specific logic

EPUB 2 and EPUB 3 differ in meaningful ways, and the split should be explicit in the code rather than smeared across conditionals. When adding parsing logic that behaves differently between versions:

- Read the version attribute early. Error on unrecognised values — do not silently fall back to a default.
- Expose public `V2` / `V3` entry points (e.g. `DecodePackageV2` / `DecodePackageV3`) so callers who already know the version can bypass auto-detection, or try themselves when we return an error.
- Dispatch to a version-specific private function via a `switch` on the major version number (e.g. `extractMetadataV2` / `extractMetadataV3`).
- Factor shared logic into a common helper rather than duplicating it in each branch.
- Keep version-agnostic concerns — manifest assembly, spine assembly, container parsing — outside the version switch entirely.

See `opf.go` for the reference implementation of this pattern.
