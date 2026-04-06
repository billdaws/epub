# Contributing

## Setup

1. Install git hooks:

```
make setup
```

  - Optional - Setup a [Go workspace](https://go.dev/doc/tutorial/workspaces) to do manual tests with a real project, or write scripts.

1. Review existing documentation and standards.
1. Iterate on your change and test it.
1. Open a pull request and assign it to `@billdaws`.

## Commit messages

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification, enforced by a `commit-msg` hook installed via `make setup`.

**Format:**

```
type(scope)?: description
```

- **type** — one of: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`
- **scope** — optional, free-form, e.g. `opf`, `ncx`, `zip`
- **description** — lower-case, no trailing period, imperative mood

**Rules enforced by the hook:**

- Header must not exceed 100 characters.
- Description must begin with a lowercase letter.
- Description must not end with a period.
- Body (if present) must be separated from the header by a blank line.
- Body lines must not exceed 100 characters.

**Breaking changes** — append `!` before the colon: `feat(opf)!: remove deprecated field`.

**Examples:**

```
feat(opf): add support for EPUB 3 metadata refines
fix: return error on empty container.xml
docs: document V2/V3 dispatch pattern
test(ncx): add fixture for malformed toc.ncx
refactor(zip): extract archive helper
```

## Design goals

**High test coverage.** Every public API should have unit tests. Where possible, use real public-domain EPUB files (e.g. from Project Gutenberg) as integration fixtures rather than synthetic ones — they catch real-world edge cases that hand-crafted test data tends to miss.

**Go idioms.** Aim for conventional, unsurprising Go. No panics. No global state. Error handling should be explicit and local. If something feels like it needs a clever abstraction, it probably just needs a clearer function name.

**Struct tags.** All externally-facing structs (those that appear in public API signatures) must have `json:"..."` tags using `snake_case` keys. Add `omitempty` where a field is intentionally optional (e.g. absent in some EPUB versions). Internal-only structs (XML helpers, unexported types) do not need JSON tags.

**Zero dependencies.** Only the Go standard library. Do not add third-party modules.

## Adding EPUB version-specific logic

EPUB 2 and EPUB 3 differ in meaningful ways, and the split should be explicit in the code rather than smeared across conditionals. When adding parsing logic that behaves differently between versions:

- Read the version attribute early. Error on unrecognised values — do not silently fall back to a default.
- Expose public `V2` / `V3` entry points (e.g. `DecodePackageV2` / `DecodePackageV3`) so callers who already know the version can bypass auto-detection, or try themselves when we return an error.
- Dispatch to a version-specific private function via a `switch` on the major version number (e.g. `extractMetadataV2` / `extractMetadataV3`).
- Factor shared logic into a common helper rather than duplicating it in each branch.
- Keep version-agnostic concerns — manifest assembly, spine assembly, container parsing — outside the version switch entirely.

See `opf.go` for the reference implementation of this pattern.
