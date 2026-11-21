# Repository history migration

On November 21, 2026, VPN-AIO's branch and tag history was separated from the
inherited history predating this project's 2.0.0 development. The 63 VPN-AIO
development commits and their merge relationships were retained. Their file
trees, author/committer identities and dates, and commit messages are unchanged.
Parent identifiers changed, so commit identifiers changed and the three old
GitHub merge signatures no longer apply. No replacement signatures are claimed.

The v1.x tags and the inherited `feat/ssl-manager-and-inbound-fixes` branch were
removed. The project's four development branches and the following releases
remain. Each release tag has the exact same Git tree as before the migration.

| Release | Original tag commit | Rewritten tag commit |
| --- | --- | --- |
| v2.0.0 | `787bb486d173099e655b561e400be748c99c6179` | `a6912196ef692b652ca7f5938ba4dd041b9d793c` |
| v2.1.0 | `b84bcaa159b6877220720bdfcb9c68a66571de63` | `8056fdb8bf72da6b9aba286dc41ee9cddccf1223` |
| v3.0.0 | `ba06be54228736b06c5da640bcb0424dfed7456d` | `665aa13f6e75886cc194d5780b161f8789060be8` |

## Published binaries and source provenance

The existing release assets, checksums, embedded-input inventories and build
manifests were retained without modification. They describe the original builds;
do not rewrite their recorded source identifiers or claim that the binaries were
rebuilt. `history-commit-map.json` maps every retained original commit to a new
commit with exactly the same file tree, including the build-source commits
mentioned in release documentation and manifests. The one subsequent migration
documentation commit adds only this file and that mapping to `main`.

## Continuing development

This workspace's local development branches were migrated along with the remote.
Other pre-migration clones must not merge or push their old branches into the
rewritten repository: doing so can restore the inherited history. Save any
uncommitted work first, then use a fresh clone. Transfer unpublished work as a
reviewed patch or cherry-pick it onto the corresponding rewritten branch rather
than merging the old history. Do not push an old mirror or use `git push --mirror`.

The pinned third-party submodules, licenses, copyright notices, upstream credits,
installer code, release download names and version selectors were not changed.
This history boundary does not change the upstream origin or licensing of code.

## Recovery and GitHub display

A verified complete pre-migration Git bundle, original mirror, original release
metadata and the migration report were saved outside this repository in the
local `VPN-AIO-history-backup-20261121` sibling directory. Keep that directory
until recovery is no longer needed. Inspect the bundle in a separate local clone;
restoring it to GitHub would also restore the old history and contributor list.

GitHub may retain old commit objects and pull-request references and may cache
contributor statistics. The migration changes normal branch/tag history; it does
not claim that old objects or pull-request records have been erased from GitHub.
