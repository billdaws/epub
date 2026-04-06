# epub — Go Library for EPUB Files

## Design Goals

1. **High test coverage.** Unit tests for all public APIs and integration tests using real public-domain books (e.g. from Project Gutenberg).
2. **Go idioms.** Unsurprising, conventional Go: clear error handling, no global state, errors as values, zero use of `panic` in library code.
3. **Zero dependencies.** Only the Go standard library. No third-party modules.
