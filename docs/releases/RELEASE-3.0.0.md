# VPN-AIO 3.0.0

Managed Hosts add reusable, per-inbound endpoint policies to the existing Vue panel.

- Dedicated Vue Hosts page, group editor, bulk enable/delete and ordering.
- Per-Host address/port, TLS/Reality, SNI pools, transport and client overrides.
- CIDR endpoint generation and per-format exclusions.
- Per-account Host selection with scoped fail-closed deleted bindings.
- RAW, JSON and Clash rendering with legacy endpoint migration/fallback.
- Existing account credentials, device addresses and quota accounting preserved.
- Native client exports and inbound backup/import integration.
- Admin/inbound authorization on all Host management routes.

See [HOSTS.md](../guides/HOSTS.md) for semantics, API, compatibility and rollback behavior, and
[`hosts-plan.md`](../plans/hosts-plan.md) for the implementation and validation record.

### Upgrade and rollback

Back up the application directory and SQLite database before upgrading. Hosts
migration retains the legacy inline endpoint lists; an older binary does not
understand Hosts-only changes. Restore the matching pre-upgrade database when an
exact rollback is required. Published v2.0.0 and v2.1.0 assets are retained.

### Compatibility

- MASQUE continues to use supported JSON subscriptions; no raw share URI is added.
- Mux/Sockopt apply to supported JSON clients; advanced fields are not fabricated
  for unsupported subscription formats.
- The pinned Xray core removed `allowInsecure`. That switch is offered only for
  compatible RAW/Clash clients. JSON uses supported certificate pins and
  verification names, with pins serialized as the required string.
- Existing account credentials, per-device keys, quota and traffic identity are
  preserved. Group edits and imports retain/remap account Host bindings.

Release assets include `VPN-AIO-amd64`, `SHA256SUMS`, `BUILD-MANIFEST.json` and the
embedded-input checksum inventory. Verify the downloaded binary with SHA-256
before installation. See [the validation record](validation/VPN-AIO-3.0.0.md) for build provenance and release gates.

`VPN-AIO-amd64`: **354,573,744 bytes**

```text
4261971d3bcde8d95254fa4b4b2d59328619cd2e2fe8af0775809e60355c0603  VPN-AIO-amd64
```

Validation passed: the complete Linux Go suite, pinned-core Host JSON acceptance,
three Node regression scripts, browser form/link checks and 30/30 isolated
Ubuntu-container lifecycle checks. The exact release binary also passed 8/8
existing-install upgrade/HTTPS/Host-schema/rollback checks; the original test
server's binary/database/unit and service health were restored. See the
validation record for exact provenance and limits.

The cross-distribution VM/kernel/DKMS matrix remains explicitly waived. That
waiver is a coverage limitation and must not be described as a passing test.
