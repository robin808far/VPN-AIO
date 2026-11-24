package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/mhsanaei/3x-ui/v2/backend"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
)

// SingboxService manages the sing-box core: one process per singbox inbound,
// each with its own config.json under /etc/vpn-ui-singbox/server-<id>.
//
// A singbox inbound stores the STANDARD sing-box inbound object (type, listen,
// listen_port, users, tls, transport, ...) directly as its Settings JSON — the
// panel adds no wrapper, so anything sing-box accepts in its "inbounds" array
// works verbatim. Validation runs `sing-box check` on every render before a
// start; a check failure is logged and the process is not launched.
type SingboxService struct{}

var singboxService = &SingboxService{}

func singboxProcName(inboundId int) string {
	return fmt.Sprintf("singbox-server-%d", inboundId)
}

// configDir holds this inbound's config.json.
func (s *SingboxService) configDir(inboundId int) string {
	return fmt.Sprintf("/etc/vpn-ui-singbox/server-%d", inboundId)
}

func (s *SingboxService) binaryPath() string {
	return daemonBin("sing-box")
}

// Available reports whether the sing-box binary is bundled for this arch (or
// present on PATH, so a source install still works).
func (s *SingboxService) Available() bool {
	if backend.DaemonPath("sing-box") != "" {
		return true
	}
	_, err := exec.LookPath("sing-box")
	return err == nil
}

// singboxOnlyProtocols are the protocols ONLY sing-box can serve (no Xray
// counterpart exists even in the patched core); anything else is sing-box just
// when the inbound's Core field says so.
var singboxOnlyProtocols = []string{"shadowtls"}

// GetSingboxInbounds returns every inbound the sing-box core serves: the
// sing-box-only protocols plus any Xray-family protocol whose Core is "singbox".
func (s *SingboxService) GetSingboxInbounds() ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	db := database.GetDB()
	err := db.Model(model.Inbound{}).
		Where("protocol IN ? OR (core = ? AND protocol NOT IN ?)",
			singboxOnlyProtocols, "singbox", singboxOnlyProtocols).
		Find(&inbounds).Error
	return inbounds, err
}

// buildConfig renders one inbound's full sing-box config: the stored inbound
// object as the only inbound, plus a direct outbound (sing-box requires at
// least one outbound to start).
func (s *SingboxService) buildConfig(inbound *model.Inbound) ([]byte, error) {
	var block json.RawMessage
	if err := json.Unmarshal([]byte(inbound.Settings), &block); err != nil {
		return nil, fmt.Errorf("inbound %d: settings is not a sing-box inbound object: %w", inbound.Id, err)
	}
	baseText, err := baseConfigService.GetBaseConfig()
	if err != nil {
		return nil, err
	}
	return s.mergeIntoBase(block, baseText)
}

// checkConfig validates a rendered config with `sing-box check`; it is a no-op
// pass when the binary is not installed (the start would fail loudly anyway).
func (s *SingboxService) checkConfig(path string) error {
	bin := s.binaryPath()
	if !s.Available() {
		return nil
	}
	cmd := exec.Command(bin, "check", "-c", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sing-box check failed: %s", bytes.TrimSpace(out))
	}
	return nil
}

// generateServerConfig renders and writes one inbound's config.json. It reports
// whether the bytes changed, so the caller can restart that one process.
func (s *SingboxService) generateServerConfig(inbound *model.Inbound) (bool, error) {
	dir := s.configDir(inbound.Id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}
	content, err := s.buildConfig(inbound)
	if err != nil {
		return false, err
	}
	content = []byte(applyCoreConfigOverride("singbox", inbound.Id, "config.json", string(content)))
	path := dir + "/config.json"
	old, readErr := os.ReadFile(path)
	if readErr == nil && bytes.Equal(old, content) {
		return false, nil
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

// restartServer (re)starts one inbound's sing-box after validating its config.
func (s *SingboxService) restartServer(inbound *model.Inbound) {
	dir := s.configDir(inbound.Id)
	name := singboxProcName(inbound.Id)
	if err := s.checkConfig(dir + "/config.json"); err != nil {
		logger.Warning("sing-box: inbound", inbound.Id, "config check failed, not starting:", err)
		return
	}
	if err := procMgr.Start(name, s.binaryPath(), []string{"run", "-c", dir + "/config.json"}, nil, dir); err != nil {
		logger.Warning("sing-box: restart of", name, "failed:", err)
	}
}

// InitSingbox brings the sing-box stack up at panel start.
func (s *SingboxService) InitSingbox() {
	inbounds, err := s.GetSingboxInbounds()
	if err != nil || len(inbounds) == 0 {
		return
	}
	if err := s.GenerateAllConfigs(); err != nil {
		logger.Warning("sing-box: config generation failed:", err)
		return
	}
	if err := s.RestartServices(); err != nil {
		logger.Warning("sing-box: failed to start services:", err)
	}
}

// RestartServices starts or restarts every enabled singbox inbound, honouring
// maintenance mode: during maintenance this is a no-op so the operator's stop
// holds.
func (s *SingboxService) RestartServices() error {
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		logger.Info("sing-box: maintenance active, not starting inbounds")
		return nil
	}
	return s.restartServices()
}

// GenerateAllConfigs renders every enabled singbox inbound's config.json.
func (s *SingboxService) GenerateAllConfigs() error {
	inbounds, err := s.GetSingboxInbounds()
	if err != nil {
		return err
	}
	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		changed, err := s.generateServerConfig(inbound)
		if err != nil {
			logger.Warning("sing-box: config for inbound", inbound.Id, "failed:", err)
			continue
		}
		if changed && procMgr.IsRunning(singboxProcName(inbound.Id)) {
			s.restartServer(inbound)
		}
	}
	return nil
}

// restartServices starts every enabled singbox inbound and stops processes for
// inbounds that vanished or were disabled. sing-box has no hot reload: any
// config change is a restart of that inbound only. Callers go through the
// exported RestartServices, which honours maintenance mode.
func (s *SingboxService) restartServices() error {
	inbounds, err := s.GetSingboxInbounds()
	if err != nil {
		return err
	}
	desired := map[string]bool{}
	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		dir := s.configDir(inbound.Id)
		if _, err := s.generateServerConfig(inbound); err != nil {
			logger.Warning("sing-box: config for inbound", inbound.Id, "failed:", err)
			continue
		}
		name := singboxProcName(inbound.Id)
		desired[name] = true
		if err := s.checkConfig(dir + "/config.json"); err != nil {
			logger.Warning("sing-box: inbound", inbound.Id, "config check failed, not starting:", err)
			continue
		}
		if err := procMgr.Start(name, s.binaryPath(), []string{"run", "-c", dir + "/config.json"}, nil, dir); err != nil {
			logger.Warning("sing-box: failed to start", name, err)
		}
	}
	for _, name := range procMgr.namesWithPrefix("singbox-server-") {
		if !desired[name] {
			_ = procMgr.Stop(name)
		}
	}
	return nil
}

// EnsureServicesRunning starts any enabled singbox inbound that has no running
// process, leaving running instances untouched. Maintenance mode holds them
// down: the operator's deliberate stop is not undone by panel starts or
// inbound saves.
func (s *SingboxService) EnsureServicesRunning() error {
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		return nil
	}
	inbounds, err := s.GetSingboxInbounds()
	if err != nil {
		return err
	}
	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		name := singboxProcName(inbound.Id)
		if procMgr.IsRunning(name) {
			continue
		}
		dir := s.configDir(inbound.Id)
		if _, err := s.generateServerConfig(inbound); err != nil {
			logger.Warning("sing-box: config for inbound", inbound.Id, "failed:", err)
			continue
		}
		if err := s.checkConfig(dir + "/config.json"); err != nil {
			logger.Warning("sing-box: inbound", inbound.Id, "config check failed, not starting:", err)
			continue
		}
		if err := procMgr.Start(name, s.binaryPath(), []string{"run", "-c", dir + "/config.json"}, nil, dir); err != nil {
			logger.Warning("sing-box: failed to start", name, err)
		}
	}
	return nil
}

// StopServices stops all sing-box child processes.
func (s *SingboxService) StopServices() error {
	procMgr.StopByPrefix("singbox-server-")
	return nil
}

// SetupRouting is a no-op: sing-box is a userspace relay with no tunnel, so
// there are no nftables rules or kernel modules to install. Kept so the service
// matches the shape of its siblings.
func (s *SingboxService) SetupRouting() error { return nil }
