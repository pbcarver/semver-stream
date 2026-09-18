# semver-stream

A semantic version parser and pretty printer for Go, built around the
case that most libraries in this space skip: validating a large stream of
version strings (a git tag dump, a registry export, a build log) without
reading the whole thing into memory first.

## Why

Most semver packages take a `string` and return a `Version`. That's fine
for the "check this one tag" case. It falls over when the input is a
million-line file: you either load it all up front, or you write your own
line-by-line loop and hope you got the buffering right. This package
makes the streaming path a first-class citizen instead of an exercise
left to the caller.

## Usage

Parsing and printing a single version:

```go
v, err := semver.Parse("1.4.2-rc.1+build.77")
if err != nil {
    log.Fatal(err)
}
fmt.Println(v.Major, v.Minor, v.Patch) // 1 4 2
fmt.Println(v.String())                // 1.4.2-rc.1+build.77
```

Validating a stream without buffering the whole input:

```go
f, err := os.Open("tags.txt")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

sc := semver.NewScanner(f)
for sc.Next() {
    res := sc.Result()
    if res.Err != nil {
        fmt.Printf("line %d: invalid: %v\n", res.Line, res.Err)
        continue
    }
    fmt.Printf("line %d: %s\n", res.Line, res.Version)
}
if err := sc.Err(); err != nil {
    log.Fatal(err)
}
```

`Scanner` reads one line at a time with a `bufio.Reader`, so memory use is
bounded by the longest single line in the input, not by the size of the
file.

## CLI

`cmd/semvercheck` is a thin wrapper around the same scanner:

```sh
go run ./cmd/semvercheck tags.txt
go run ./cmd/semvercheck < tags.txt
```

It prints the canonical form of every valid line to stdout and every
parse error to stderr, and exits non-zero if anything failed to parse.

## Scope

This is a strict implementation of the grammar at semver.org:

- No `v` prefix (`v1.2.3` is rejected; strip it yourself if your source
  uses it).
- No leading zeros in numeric identifiers.
- No whitespace, no empty identifiers.

Comparison and sorting aren't implemented yet — see below.

## Status

First pass. The parser and streaming scanner work and are tested against
the semver.org examples, but the library doesn't do everything yet.
