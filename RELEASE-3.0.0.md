# VPN-AIO 3.0.0 — development, not published

Managed Hosts are implemented and locally validated on `feature/hosts-v3.0.0`.

- Dedicated Vue Hosts page, group editor, bulk enable/delete and ordering.
- Per-Host address/port, TLS/Reality, SNI pools, transport and client overrides.
- CIDR endpoint generation and per-format exclusions.
- Per-account Host selection with scoped fail-closed deleted bindings.
- RAW, JSON and Clash rendering with legacy endpoint migration/fallback.
- Existing account credentials, device addresses and quota accounting preserved.
- Native client exports and inbound backup/import integration.
- Admin/inbound authorization on all Host management routes.

See `HOSTS.md` for semantics, API, compatibility and rollback behavior, and
`hosts-plan.md` for the implementation and validation record.

No v3.0.0 tag or public release has been created. The published v2.0.0 and v2.1.0
releases are unchanged. A final release asset, checksum and publication decision
follow validation; this document does not identify a released binary.

Validation passed: the complete Linux Go suite, pinned-core Host JSON acceptance,
three Node regression scripts, browser form/link checks and 30/30 isolated
Ubuntu-container lifecycle checks. The original test-server service was not
replaced. See the plan's validation record for exact provenance and limits.

The cross-distribution VM/kernel/DKMS matrix remains explicitly waived. That
waiver is a coverage limitation and must not be described as a passing test.
