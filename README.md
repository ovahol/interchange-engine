# interchange-engine

The pure engine behind the [Medical Device Lifecycle Interchange](../../docs/interchange-api.md)
pipeline: ingest normalization, taxonomy classification, catalog matching, and
identity/vocabulary reconciliation.

This module is incubated inside the Ovahol repository and is intended to move to
`github.com/ovahol/interchange-engine` once the migration is complete. The
module path is already final, so extraction does not rewrite imports.

## Boundary

The engine is deliberately dependency-free. It must not import
`github.com/ovahol/ovahol` (enforced by
`match/architecture_test.go`), and it owns no persistence:

- **In:** normalizing headers/cells, folding and stemming names, ranking
  candidates, reconciling identity fields, mapping status vocabulary.
- **Out:** database access, tenant context, recommendation policy, bundle
  extensions, HTTP handlers, and lifecycle writes.

Callers supply candidate sets through plain data; the engine scores and decides.

## Status

Milestones 0–4 complete: the matcher (`match`), ingest layer (`ingest`), shared
text primitives (`text`), the single status vocabulary (`vocab`), taxonomy
classification (`classify`), and the single-import-route cutover (interchange
is now the only facility import path; `facilityimport` and `mdis` are gone)
are all in place. Remaining: extraction to its own repository.
