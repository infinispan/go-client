# Documentation Instructions

## Content Organization

Documentation is written in AsciiDoc format in the `documentation/` directory.

- **`infinispan-go-client.adoc`** — Main user guide covering all client features.

## Writing Style

### Voice and Tense
- **Active voice**, present tense, second person ("you").
- Never use first person ("we", "I").
- No contractions ("do not", not "don't").
- American English spelling.

### Formatting
- One sentence per line (hard wrap at sentence boundaries, not at column width).
- File paths, function names, type names, and package names: backticks (`` ` ``).
- Numbers below 10: spell out ("four"); 10 and above: numerals ("12").
- Avoid Latin abbreviations (use "for example" not "e.g.", "that is" not "i.e.").

### Code Blocks
Use `[source,go]` for Go examples and `[source,shell]` for CLI commands:

```asciidoc
[source,go]
----
client, err := hotrod.NewClient(ctx, "hotrod://admin:password@localhost:11222")
----
```

### Admonitions
```asciidoc
[NOTE]
====
Note content.
====
```
Use `NOTE`, `TIP`, `WARNING`, `IMPORTANT` as appropriate.

## Terminology

- **Cache** — the remote data structure; do not use "map" or "store" interchangeably.
- **Near cache** — local cache backed by server bloom filter invalidation; always two words.
- **Hot Rod** — two words, both capitalized; the binary protocol name.
- **Ickle** — the query language name; always capitalized.
- **ProtoStream** — one word, camel case; the protobuf serialization format.
- **Put/Get/Remove** — use these verbs consistently for cache operations.
- **Add/Remove** — for listeners and continuous queries.

## When Creating New Documentation
1. Determine where the content fits in the existing guide structure.
2. Add a new section in `infinispan-go-client.adoc` under the appropriate heading.
3. Include working Go code examples that compile and run.
4. Mark incomplete sections with `// TODO:` comments.
