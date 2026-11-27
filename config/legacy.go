package config

import (
	"os"
	"strings"
)

// This file is the ONLY place that names the product's previous identity. The
// panel was renamed from that name to VPN-AIO in 7.0.1; everything below exists
// so an install made before the rename keeps working and is moved over
// automatically (see service.MigrateLegacyLayout). Remove it, together with the
// migration, once no supported install can still carry the old layout.

const (
	// LegacyToken is the old kebab-case product name used in paths, files and
	// commands; CurrentToken is what replaces it.
	LegacyToken  = "vpn-ui"
	CurrentToken = "vpn-aio"

	// legacyEnvPrefix / currentEnvPrefix are the environment-variable prefixes.
	// The legacy prefix is still read for one release as a fallback.
	legacyEnvPrefix  = "VPNUI_"
	currentEnvPrefix = "VPNAIO_"
)

// LegacyScanDirs are the host directories whose direct entries may carry the
// legacy token in their name (/etc/vpn-ui-singbox, /usr/libexec/vpn-ui-accel,
// /etc/sysctl.d/99-vpn-ui.conf, /usr/bin/vpn-ui, ...). The migration renames
// every such entry by substituting CurrentToken for LegacyToken.
var LegacyScanDirs = []string{
	"/etc",
	"/etc/modules-load.d",
	"/etc/sysctl.d",
	"/etc/strongswan.d",
	"/etc/swanctl/x509ca",
	"/usr/libexec",
	"/usr/src",
	"/usr/local",
	"/var/log",
	"/run",
}

// LegacyProcessPattern is a `pkill -f` pattern that matches child daemons a
// previous panel started from the legacy runtime dirs or bundles (for example
// `sing-box run -c /etc/vpn-ui-singbox/server-5/config.json`).
const LegacyProcessPattern = "/(etc|usr/libexec)/" + LegacyToken

// LegacyMenuPath is the old management-menu command. The new one is installed
// by ensureMenuInstalled, so the old file is simply removed.
const LegacyMenuPath = "/usr/bin/" + LegacyToken

// LegacySystemdUnit is the old default unit name. It is never removed while it
// is the running unit; the migration only warns about it.
const LegacySystemdUnit = LegacyToken + ".service"

// LegacyName replaces CurrentToken with LegacyToken in a base name, giving the
// pre-rename name of a file the panel now calls name (vpn-aio.sock -> old).
func LegacyName(name string) string {
	return strings.ReplaceAll(name, CurrentToken, LegacyToken)
}

// CurrentName is the reverse of LegacyName.
func CurrentName(name string) string {
	return strings.ReplaceAll(name, LegacyToken, CurrentToken)
}

// HasLegacyToken reports whether s still carries the legacy name.
func HasLegacyToken(s string) bool {
	return strings.Contains(s, LegacyToken)
}

// getenv reads a VPNAIO_* variable and falls back to its legacy-prefixed twin,
// so an old .env or systemd Environment= line keeps working for one release.
func getenv(name string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	if strings.HasPrefix(name, currentEnvPrefix) {
		return os.Getenv(legacyEnvPrefix + strings.TrimPrefix(name, currentEnvPrefix))
	}
	return ""
}
