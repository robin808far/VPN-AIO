package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
)

// singboxBaseDir is the shared scratch/root directory for sing-box files.
var singboxConfigRoot = "/etc/vpn-ui-singbox"

func singboxBaseDir() string {
	return singboxConfigRoot
}

// osWriteFile is a thin alias so the intent at call sites stays clear.
func osWriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// BaseConfigService stores the operator's sing-box BASE config — the top-level
// blocks around the per-inbound objects the panel generates: log, dns, ntp,
// route, experimental (cache file, clash api) and shared outbounds. It is the
// analogue of s-ui's Basics/DNS/Rules editors: the per-inbound object stays in
// the inbound's own settings, everything a real deployment needs around it
// lives here and is merged into every rendered config.
//
// The stored text is validated with `sing-box check` before it is saved, so a
// bad block never reaches a running core.
type BaseConfigService struct{}

var baseConfigService = &BaseConfigService{}

const singboxBaseSettingKey = "singboxBaseConfig"

// DefaultSingboxBaseConfig is served when nothing is stored: the exact shape
// buildConfig produced before base editing existed.
const DefaultSingboxBaseConfig = `{
  "log": { "level": "info", "timestamp": true },
  "outbounds": [ { "type": "direct" } ]
}`

// baseConfigBlocks are the top-level keys an operator may set. New saves reject
// "inbounds" because listeners are managed per inbound; legacy stored values are
// ignored during rendering so they cannot add an unmanaged listener.
var baseConfigBlocks = []string{"log", "dns", "ntp", "route", "experimental", "outbounds", "endpoints", "http_clients"}

// GetBaseConfig returns the stored base-config JSON, or the default when no
// override exists. The returned text is always valid JSON.
func (s *BaseConfigService) GetBaseConfig() (string, error) {
	db := database.GetDB()
	var setting model.Setting
	err := db.Model(model.Setting{}).Where("key = ?", singboxBaseSettingKey).First(&setting).Error
	if err != nil || setting.Value == "" {
		return DefaultSingboxBaseConfig, nil
	}
	return setting.Value, nil
}

// SaveBaseConfig validates the text as a sing-box config skeleton (with a
// placeholder inbound and the stored blocks merged around it), stores it, and
// regenerates every singbox inbound so it lands immediately. It reports the
// validation output on failure.
func (s *BaseConfigService) SaveBaseConfig(text string) error {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	merged, err := mergeBaseConfig(text)
	if err != nil {
		return err
	}
	svc := &SingboxService{}
	if err := svc.checkConfigText(merged); err != nil {
		return err
	}
	inbounds, err := svc.GetSingboxInbounds()
	if err != nil {
		return err
	}
	groups, err := singboxGroupsFromInbounds(inbounds)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if !singboxGroupEnabled(group) {
			continue
		}
		if err := svc.ensureAPIPort(group.Root); err != nil {
			return err
		}
		runtimeMembers, err := svc.runtimeGroupMembers(group)
		if err != nil {
			return err
		}
		content, err := svc.renderConfigWithBase(group.Root, runtimeMembers, text)
		if err != nil {
			return err
		}
		if err := svc.checkConfigText(content); err != nil {
			return fmt.Errorf("inbound %d with proposed base: %w", group.Root.Id, err)
		}
	}
	db := database.GetDB()
	var setting model.Setting
	err = db.Model(model.Setting{}).Where("key = ?", singboxBaseSettingKey).First(&setting).Error
	if err != nil {
		setting = model.Setting{Key: singboxBaseSettingKey, Value: text}
		return db.Create(&setting).Error
	}
	setting.Value = text
	return db.Save(&setting).Error
}

// ApplyBaseConfig regenerates every singbox inbound against the stored base
// config, restarting the processes whose bytes changed.
func (s *BaseConfigService) ApplyBaseConfig() error {
	svc := &SingboxService{}
	return svc.GenerateAllConfigs()
}

// mergeBaseConfig folds the stored base blocks into the minimal skeleton
// buildConfig renders for an inbound, so validation sees the exact shape a
// running core would see.
func mergeBaseConfig(baseText string) ([]byte, error) {
	var base map[string]json.RawMessage
	if err := json.Unmarshal([]byte(baseText), &base); err != nil {
		return nil, fmt.Errorf("base config is not a JSON object: %w", err)
	}
	if base == nil {
		return nil, fmt.Errorf("base config must be a JSON object")
	}
	if err := rejectBaseConfigInbounds(base); err != nil {
		return nil, err
	}
	cfg := map[string]any{
		"log": map[string]any{"level": "info", "timestamp": true},
		"inbounds": []map[string]any{
			{"type": "direct", "tag": "placeholder", "listen": "127.0.0.1", "listen_port": 1},
		},
		"outbounds": []map[string]any{{"type": "direct"}},
	}
	for _, key := range baseConfigBlocks {
		if raw, ok := base[key]; ok {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("block %q is not valid JSON: %w", key, err)
			}
			cfg[key] = value
		}
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// mergeIntoBase merges the stored base blocks into one inbound's rendered
// config: the base wins for every block it defines, the inbound object is
// always the only entry of "inbounds".
func (s *SingboxService) mergeIntoBase(inboundObj json.RawMessage, baseText string) ([]byte, error) {
	return s.mergeIntoBaseInbounds([]json.RawMessage{inboundObj}, baseText)
}

func (s *SingboxService) mergeIntoBaseInbounds(inboundObjects []json.RawMessage, baseText string) ([]byte, error) {
	var base map[string]json.RawMessage
	if err := json.Unmarshal([]byte(baseText), &base); err != nil {
		// An unreadable stored base falls back to the generated default rather
		// than taking every singbox inbound down.
		logger.Warning("sing-box: stored base config unreadable, using defaults:", err)
		base = nil
	}
	cfg := map[string]any{
		"log":       map[string]any{"level": "info", "timestamp": true},
		"inbounds":  inboundObjects,
		"outbounds": []map[string]any{{"type": "direct"}},
	}
	for _, key := range baseConfigBlocks {
		if raw, ok := base[key]; ok {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				continue
			}
			cfg[key] = value
		}
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func rejectBaseConfigInbounds(base map[string]json.RawMessage) error {
	if _, exists := base["inbounds"]; exists {
		return fmt.Errorf(`base config must not contain "inbounds"; listeners are configured per inbound`)
	}
	return nil
}

// checkConfigText uses a unique, private file even on a fresh installation.
func (s *SingboxService) checkConfigText(content []byte) error {
	dir := singboxBaseDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Clean(dir), ".check-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return s.checkConfig(file.Name())
}
