# Backlog

- [ ] **Parse EPUB container** — open an `.epub` file (ZIP), locate `META-INF/container.xml`, and resolve the root OPF document path.
- [ ] **Parse OPF package** — read `content.opf` to extract metadata (title, author, language, identifier, publication date), the manifest (all items with id/href/media-type), and the spine (reading order).
- [ ] **Parse NCX / Navigation Document** — parse `toc.ncx` (EPUB 2) and `nav.xhtml` (EPUB 3) to expose a structured table of contents tree.
- [ ] **Read content items** — given a manifest item, return the raw bytes for its content so callers can decode HTML, images, CSS, etc.
- [ ] **Write / create EPUB** — build a valid EPUB 3 file from scratch: add metadata, add content items to the manifest, define spine order, and write to a file.
- [ ] **Validate structure** — check that a parsed EPUB conforms to required EPUB 3 (and EPUB 2 compat) structural rules and report violations as typed errors.
