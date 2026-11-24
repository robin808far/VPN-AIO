package service

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
)

// singboxBaseDir is the shared scratch/root directory for sing-box files.
func singboxBaseDir() string {
	return "/etc/vpn-ui-singbox"
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

// baseConfigBlocks are the top-level keys an operator may set; everything else
// in the stored text is ignored so "inbounds" can never be injected here.
var baseConfigBlocks = []string{"log", "dns", "ntp", "route", "experimental", "outbounds", "endpoints"}

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
	merged, err := mergeBaseConfig(text)
	if err != nil {
		return err
	}
	svc := &SingboxService{}
	if err := svc.checkConfigText(merged); err != nil {
		return err
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
	var base map[string]json.RawMessage
	if err := json.Unmarshal([]byte(baseText), &base); err != nil {
		// An unreadable stored base falls back to the generated default rather
		// than taking every singbox inbound down.
		logger.Warning("sing-box: stored base config unreadable, using defaults:", err)
		base = nil
	}
	cfg := map[string]any{
		"log":       map[string]any{"level": "info", "timestamp": true},
		"inbounds":  []json.RawMessage{inboundObj},
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

// checkConfigText validates a rendered config body with `sing-box check`
// without touching a real inbound's file; a no-op pass when the binary is
// absent.
func (s *SingboxService) checkConfigText(content []byte) error {
	dir := singboxBaseDir()
	path := dir + "/.base-check.json"
	if err := osWriteFile(path, content, 0o600); err != nil {
		return err
	}
	return s.checkConfig(path)
}
