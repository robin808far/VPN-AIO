# Managed Hosts — VPN-AIO 3.0.0

Hosts are client-facing endpoint policies. They change exported configurations,
links and QR codes; they do not open a listener, configure a CDN, issue a
certificate, rotate a credential or change account traffic accounting.

## Groups and inheritance

Open **Hosts** in the sidebar. A group combines a set of addresses with a set of
inbounds. One stored row represents one address on one inbound. Groups support
editing, enable/disable, bulk operations and ordering.

Enter one domain, IP or CIDR per line. A line may include a port. Write IPv6 with
an explicit port as `[2001:db8::10]:8443`. An empty address inherits the inbound's
address, and port `0` inherits its listening port. An enabled group can therefore
override just SNI, a transport path or another client option.

Hostless inbounds retain the existing external-proxy behavior. When enabled
Hosts apply to a subscription format, those Hosts replace the legacy endpoint
list. If every Host is disabled or excluded from that format, an unbound account
uses the legacy/default path. Disabling every Host is not an inbound kill switch.

CIDR addresses expand to 1–100 distinct random addresses per Host per render,
bounded by the usable address space. IPv4 network/broadcast addresses and the
IPv6 subnet-router anycast address are omitted where applicable. Total render
expansion is bounded at 4096 endpoints; oversized output fails explicitly.

An SNI list chooses an entry per endpoint. Shared random SNI chooses one entry
per group for one subscription request. Address-as-SNI and explicitly blank SNI
are mutually exclusive. Empty ordinary SNI inherits; the explicit blank switch
clears it. Selecting `reality` requires a Reality inbound: Hosts do not invent
Reality keys. Switching a Reality export to TLS removes server Reality settings
from the client view.

## Output capabilities

| Setting | RAW links / panel QR | Xray JSON | Clash / Mihomo |
| --- | --- | --- | --- |
| Address, port, remark | Where that protocol has a link | Supported Xray outbounds | Supported Clash proxies |
| SNI, fingerprint, ALPN | Where the URI supports them | Yes | Where the proxy supports them |
| Host header / path | WS, HTTPUpgrade, XHTTP | Matching transports | Matching supported transports |
| Mux / Sockopt | No general URI equivalent | Yes | Not converted |
| FinalMask | Compatible links/clients | Merges TCP/UDP masks; inherits existing QUIC parameters | Not converted |
| XHTTP extras | Compatible XHTTP links | XHTTP stream settings | Subject to existing client support |
| VLESS route | Rewrites the exported UUID route bytes | Same | Same |
| Mihomo IP version / X25519 | X25519 hint where supported | No Mihomo-only conversion | Yes |

Unsupported formats remain unsupported. In particular MASQUE continues to use
its supported JSON subscription; Hosts do not add a fabricated MASQUE share URI.
TLS verification, pinning and ECH options also require support in the receiving
client. `allowInsecure` is opt-in for compatible RAW/Clash clients only. The pinned
Xray core removed that flag: JSON retains verification and uses the supported
`pinnedPeerCertSha256` comma-separated string and `verifyPeerCertByName` options.

Host policy also reaches native WireGuard/AWG and SSH configuration exports and
OpenVPN profiles. Credential material and per-device addresses remain unchanged.
GRE and inbound types without a supported client endpoint export are not offered
as new Host targets. This does not alter their existing configuration downloads.

Optional name templates use double braces around `INBOUND`, `EMAIL`/`USERNAME`,
`ADDRESS`, `PORT`, `TRAFFIC_LEFT`, `TRAFFIC_USED`, `TRAFFIC_TOTAL` or `DAYS_LEFT`.
The existing remark model remains in use for ordinary names. English and Persian
Hosts labels are provided; the other existing locales carry explicit English
fallback text for the new section.

## Account selection

The account editor's **Limits** tab contains **Host selection**. Empty selection
uses each inbound's default Hosts. A nonempty selection restricts the inbounds
the selected Hosts belong to; unrelated inbound memberships keep their defaults.

Disabled, format-excluded or deleted selected Hosts do not cause a restricted
account to fall back to a different endpoint on the affected inbound. Deletions
retain an inbound-scoped tombstone for this purpose. Unknown binding IDs fail
closed. A client whose output is empty still retains its usage/account identity.

Group edits reuse row IDs for unchanged endpoints and, where possible, for
replacement endpoints on the same inbound. A remark, SNI or address edit should
therefore not discard account selections. Removing an inbound/address from a
group can intentionally remove a selected endpoint.

Selections are account-wide and are projected into membership client entries.
Partial account writes preserve them; an explicit empty array clears them.
Resellers cannot replace the administrator's managed Host selection.

## Permissions

- Reading the Hosts management API/page requires inbound access and an admin role.
- Mutating Host groups requires `editInbound`.
- Every referenced inbound must exist and be granted to the caller.
- Editing/removing/disabling a shared group requires access to every inbound in
  the existing group and every inbound in its replacement.
- A partially visible group is listed read-only without disclosing foreign
  inbound IDs. Bulk operations authorize all targets before applying a change.
- Changing panel-wide subscription sorting additionally requires panel-settings
  permission. Resellers cannot manage shared Host policy.

## Migration, import and rollback

Startup adds `hosts`, `host_tombstones` and the account Host-selection column.
It copies legacy inbound endpoint lists into Hosts once, recording
`HostsFromExternalProxyV3` in seeder history. Protocol-specific endpoint lists
inside settings are recognized alongside Xray stream settings. Existing inline
configuration is retained so the older binary can still read it on rollback.

New legacy imports are converted even after the startup migration has run.
Editing an old inline endpoint list synchronizes its untouched migrated Hosts.
Explicitly edited groups are detached from that legacy synchronization. Restart
does not recreate a Host the administrator removed.

Inbound exports carry Host rows and inbound-scoped selection metadata. Imports
allocate fresh group/row IDs and remap account bindings. Stale source IDs must
not accidentally select unrelated local Hosts; unavailable selected endpoints
are represented by new tombstones. A full database backup preserves all IDs.

Rolling back the binary restores the old external-proxy behavior; new Hosts-only
edits are additive data the older binary does not understand. Keep a pre-upgrade
database backup when an exact rollback is required.

## API

All paths below are relative to `/panel/api/hosts`, require an authenticated
panel session and use the existing `{success, msg, obj}` response convention.
Mutation requests use `Content-Type: application/json`.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/list` | Accessible groups with `editable` status |
| GET | `/raw` | Accessible rows with numeric IDs for account pickers |
| GET | `/get/:groupId` | One visible group |
| GET | `/byInbound/:inboundId` | Visible groups serving one inbound |
| GET | `/tags` | Accessible tags |
| POST | `/add`, `/bulk/add` | Create a group |
| POST | `/update/:groupId` | Replace a group's content, preserving row IDs where possible |
| POST | `/del/:groupId` | Delete a group |
| POST | `/setEnable/:groupId` | Set state with `{ "enable": true }` |
| POST | `/bulk/del` | Delete groups with `{ "ids": ["group-id"] }` |
| POST | `/bulk/setEnable` | Set group state with `ids` and `enable` |
| POST | `/reorder` | Order submitted group IDs |
| GET | `/sortMode` | `inbound-first` or `hosts-first` |
| POST | `/sortMode` | Set `{ "mode": "hosts-first" }`; requires panel-settings permission |

Example group request:

```json
{
  "remark": "CDN edge",
  "inboundIds": [1, 2],
  "hosts": ["cdn.example.com:443", "[2001:db8::10]:8443"],
  "port": 0,
  "security": "same",
  "snis": ["front.example.com"],
  "hostHeader": "origin.example.com",
  "path": "/proxy",
  "excludeFromSubTypes": [],
  "randomCount": 1
}
```

Free-JSON fields (`muxParams`, `sockoptParams`, `finalMask`,
`xhttpExtraParams`) contain serialized JSON object strings. Account selection is
the `hostRuleIds` integer array in the existing account/client mutation API.

The design and field vocabulary are adapted from MHSanaei/3x-ui at upstream
commit `5297fd3c8495d80fab97ae511553fa545c69e913`, under the project's GPL-3.0
license. Vue rendering and VPN-AIO account, access and native-export integration
are implemented for this repository; upstream React components are not bundled.
