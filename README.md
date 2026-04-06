# epub

A pure Go library with zero dependencies for reading, writing, and validating [EPUB](https://www.w3.org/publishing/epub/) files.

```
go get github.com/billdaws/epub
```

## Features

- Read EPUB 2 and EPUB 3 archives
- Parse OPF package metadata (title, authors, language, identifier, publication date)
- Parse table of contents — prefers EPUB 3 nav documents, falls back to EPUB 2 NCX
- Read manifest content items, safe for concurrent use
- Write valid EPUB 3 archives
- Validate parsed packages against structural rules

## Usage

### Reading metadata

```go
pkg, err := epub.OpenPackage("the-republic.epub")
if err != nil {
    log.Fatal(err)
}
fmt.Println(pkg.Metadata.Title)
fmt.Println(pkg.Metadata.Authors)
```

### Reading content items

`Open` keeps the archive open between reads and is safe for concurrent use.

```go
r, err := epub.Open("the-republic.epub")
if err != nil {
    log.Fatal(err)
}
defer r.Close()

for _, item := range r.Package.Manifest {
    if item.MediaType != "application/xhtml+xml" {
        continue
    }
    data, err := r.ReadItem(item)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s: %d bytes\n", item.Href, len(data))
}
```

### Reading the table of contents

```go
toc, err := epub.OpenTOC("the-republic.epub")
if err != nil {
    log.Fatal(err)
}

var printTOC func(points []epub.NavPoint, depth int)
printTOC = func(points []epub.NavPoint, depth int) {
    for _, p := range points {
        fmt.Printf("%s%s\n", strings.Repeat("  ", depth), p.Title)
        printTOC(p.Children, depth+1)
    }
}
printTOC(toc, 0)
```

### Writing an EPUB

```go
nav := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc"><ol><li><a href="chapter1.xhtml">Chapter 1</a></li></ol></nav>
</body>
</html>`

chapter := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<body><h1>Chapter 1</h1><p>It was a dark and stormy night.</p></body>
</html>`

book := epub.Book{
    Metadata: epub.Metadata{
        Title:      "My Book",
        Language:   "en",
        Identifier: "urn:uuid:1234-5678",
        Authors:    []string{"Jane Doe"},
    },
    Items: []epub.ContentItem{
        {ID: "nav", Href: "nav.xhtml", MediaType: "application/xhtml+xml", Properties: "nav", Content: []byte(nav)},
        {ID: "ch1", Href: "chapter1.xhtml", MediaType: "application/xhtml+xml", Content: []byte(chapter)},
    },
    Spine: []string{"ch1"},
}

f, err := os.Create("output.epub")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

if err := epub.Write(f, book); err != nil {
    log.Fatal(err)
}
```

### Validating a package

```go
pkg, err := epub.OpenPackage("suspect.epub")
if err != nil {
    log.Fatal(err)
}

violations := epub.Validate(pkg)
for _, v := range violations {
    fmt.Printf("[%s] %s\n", v.Code, v.Message)
}
```

## API overview

| Function / Type                 | Description                                          |
|---------------------------------|------------------------------------------------------|
| `Open(path)`                    | Open an EPUB for repeated content reads (thread-safe)|
| `OpenPackage(path)`             | Parse OPF metadata in one call                       |
| `OpenTOC(path)`                 | Parse the table of contents                          |
| `OpenContainer(path)`           | Parse only the container (rootfile path)             |
| `DecodePackageV2(r, opfPath)`   | Decode an EPUB 2 OPF from an `io.Reader`             |
| `DecodePackageV3(r, opfPath)`   | Decode an EPUB 3 OPF from an `io.Reader`             |
| `Write(dst, book)`              | Encode a `Book` as a valid EPUB 3 archive            |
| `Validate(pkg)`                 | Check a `Package` against structural rules           |

## License

See [LICENSE](LICENSE).

## Contributing

See [CONTRIBUTING](CONTRIBUTING).
