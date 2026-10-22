# interchange-engine

The pure Go engine behind the [Medical Device Lifecycle Interchange](https://github.com/ovahol/interchange-schema)
pipeline: ingest normalization, taxonomy classification, catalog matching, and
identity/vocabulary reconciliation.

It is deliberately dependency-free (standard library only) and owns no
persistence, transport, or application types, so it can be embedded in any Go
program that needs to normalize or match medical-device data.

## Install

```bash
go get github.com/ovahol/interchange-engine@v0.1.0
```

## Packages

| Package | Responsibility |
|---|---|
| `text` | Rune folding and normalization primitives (diacritics, ligatures, legal suffixes). |
| `ingest` | Header resolution and synonyms, CSV parsing, cell normalization, loose date parsing. |
| `vocab` | The single canonical device-status / condition vocabulary and synonyms. |
| `classify` | Taxonomy loading/validation and the ordered, vendor-defined inference rule engine. |
| `match` | Token-overlap and typo-tolerant catalog/name ranking with strict-winner semantics. |

## Boundary

The engine must not import the Ovahol application (enforced by
`match/architecture_test.go`):

- **In:** normalizing headers/cells, folding and stemming names, classifying a
  device against a vendor taxonomy, ranking candidates, and mapping status
  vocabulary.
- **Out:** database access, tenant context, recommendation policy, bundle
  extensions, HTTP handlers, and lifecycle writes.

Callers supply candidate sets through plain data; the engine scores and decides.

## Development

```bash
go test ./...
go vet ./...
golangci-lint run ./...
```

## License

MIT — see [LICENSE](LICENSE).
