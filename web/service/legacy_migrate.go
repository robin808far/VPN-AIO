package service

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/config"
	"github.com/mhsanaei/3x-ui/v2/logger"
)

// MigrateLegacyLayout moves an install made before the product rename onto the
// VPN-AIO names. It must run as root at panel startup BEFORE any core, daemon,
// control socket or log consumer starts, so every generated config is written
// under the new paths. Idempotent: once nothing with the legacy name is left,
// it does nothing. The legacy names themselves live only in config/legacy.go.
func MigrateLegacyLayout() {
	if os.Geteuid() != 0 {
		return
	}
	// Stop daemons a previous panel launched from the legacy dirs/bundles first,
	// so nothing holds the trees being renamed and no orphan keeps its ports.
	if commandExists("pkill") {
		_ = exec.Command("pkill", "-KILL", "-f", config.LegacyProcessPattern).Run()
	}
	res := migrateLegacyIn("/", config.LegacyScanDirs, extraLegacyFiles())
	for _, m := range res.moved {
		logger.Infof("rename migration: %s -> %s", m[0], m[1])
	}
	for _, w := range res.warnings {
		logger.Warning("rename migration:", w)
	}
	// The old menu command: the new one is (re)installed by ensureMenuInstalled.
	if err := os.Remove(config.LegacyMenuPath); err == nil {
		logger.Info("rename migration: removed legacy menu", config.LegacyMenuPath)
	}
	if _, err := os.Stat(filepath.Join("/etc/systemd/system", config.LegacySystemdUnit)); err == nil {
		logger.Warning("rename migration: legacy systemd unit", config.LegacySystemdUnit,
			"still exists; re-run `--systemd` to install the VPN-AIO unit, then remove it")
	}
}

// MigrateLegacyLogFile moves the pre-rename panel log onto the current name. It
// must run BEFORE the logger is initialised, because the logger creates the new
// log file and the generic migration then refuses to overwrite it. When both
// names already exist (an earlier 7.0.1 start), the old log's content is
// prepended to the new one so no history is lost and the old file is removed.
// It cannot use the logger, so it reports nothing; the result is visible in the
// log file itself. Idempotent.
func MigrateLegacyLogFile() {
	if os.Geteuid() != 0 {
		return
	}
	dir := config.GetLogFolder()
	cur := filepath.Join(dir, "vpn-aio.log")
	old := filepath.Join(dir, config.LegacyName("vpn-aio.log"))
	_ = mergeLegacyFile(old, cur)
}

// mergeLegacyFile renames old to cur, or, when cur already exists, writes
// old+cur into cur and removes old. A missing old file is not an error.
func mergeLegacyFile(old, cur string) error {
	if _, err := os.Lstat(old); err != nil {
		return nil
	}
	if _, err := os.Lstat(cur); err != nil {
		return movePath(old, cur)
	}
	oldData, err := os.ReadFile(old)
	if err != nil {
		return err
	}
	curData, err := os.ReadFile(cur)
	if err != nil {
		return err
	}
	tmp := cur + ".merge"
	if err := os.WriteFile(tmp, append(oldData, curData...), 0644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, cur); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Remove(old)
}

// extraLegacyFiles lists the panel-owned files that sit next to the binary or
// in the DB/log folders and were named after the old product.
func extraLegacyFiles() []string {
	var out []string
	for _, cur := range []string{
		ControlSocketPath(),
		filepath.Join(config.GetLogFolder(), "vpn-aio.log"),
	} {
		dir, base := filepath.Split(cur)
		if legacy := config.LegacyName(base); legacy != base {
			out = append(out, filepath.Join(dir, legacy))
		}
	}
	return out
}

type legacyMigrateResult struct {
	moved    [][2]string
	warnings []string
}

// migrateLegacyIn does the work under root (a temp dir in tests). Every direct
// entry of each scan dir whose name carries the legacy token, and every file in
// extra, is renamed to the current name. A target that already exists is never
// overwritten: the legacy copy is left alone and reported.
func migrateLegacyIn(root string, scanDirs, extra []string) legacyMigrateResult {
	var res legacyMigrateResult
	var candidates []string
	for _, d := range scanDirs {
		entries, err := os.ReadDir(filepath.Join(root, d))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if config.HasLegacyToken(e.Name()) {
				candidates = append(candidates, filepath.Join(root, d, e.Name()))
			}
		}
	}
	for _, f := range extra {
		candidates = append(candidates, filepath.Join(root, f))
	}
	for _, old := range candidates {
		if _, err := os.Lstat(old); err != nil {
			continue
		}
		dst := filepath.Join(filepath.Dir(old), config.CurrentName(filepath.Base(old)))
		if dst == old {
			continue
		}
		if _, err := os.Lstat(dst); err == nil {
			// The new name already exists (e.g. an earlier panel run already
			// created it fresh). The legacy copy is not needed for the panel to
			// work, but it is set aside rather than deleted — it may still hold
			// an old log tail or a config an operator wants to inspect — and
			// renamed off the legacy token so a plain "*vpn-ui*" scan of the
			// host comes up empty once this has run.
			backup := uniqueLegacyBackupPath(dst + ".legacy-backup")
			if err := movePath(old, backup); err != nil {
				res.warnings = append(res.warnings, fmt.Sprintf("%s and %s both exist; could not set aside %s: %v", old, dst, old, err))
				continue
			}
			res.warnings = append(res.warnings, fmt.Sprintf("%s and %s both exist; kept the new one, set the old one aside as %s", old, dst, backup))
			continue
		}
		if err := movePath(old, dst); err != nil {
			res.warnings = append(res.warnings, fmt.Sprintf("could not move %s to %s: %v", old, dst, err))
			continue
		}
		res.moved = append(res.moved, [2]string{old, dst})
	}
	return res
}

// movePath renames, falling back to copy+remove across filesystems.
func movePath(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyTree(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return os.RemoveAll(src)
}

// uniqueLegacyBackupPath returns base, or base suffixed with -2, -3, ... if it
// already exists, so re-running the migration never overwrites a backup set
// aside by an earlier run.
func uniqueLegacyBackupPath(base string) string {
	if _, err := os.Lstat(base); err != nil {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, err := os.Lstat(candidate); err != nil {
			return candidate
		}
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, lerr := os.Readlink(p)
			if lerr != nil {
				return lerr
			}
			return os.Symlink(strings.ReplaceAll(link, config.LegacyToken, config.CurrentToken), target)
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case !info.Mode().IsRegular():
			return nil // sockets/fifos are recreated by their owners
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}

// removeLegacyLeftovers is the uninstall half: every pre-rename path that the
// startup migration could not move (both names existed) is removed too.
func removeLegacyLeftovers(r *UninstallReport) {
	for _, d := range config.LegacyScanDirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if config.HasLegacyToken(e.Name()) {
				removePath(r, filepath.Join(d, e.Name()))
			}
		}
	}
	removePath(r, config.LegacyMenuPath)
}
