package service

import (
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
)

// SingboxMaintenance guards the sing-box core's maintenance mode. In
// maintenance every singbox child process is stopped AND the panel's own
// restart paths (panel start, inbound save, the ensure job) leave it stopped,
// exactly like s-ui's deliberate core stop: a plain stop would otherwise be
// undone within seconds by those paths.
type SingboxMaintenance struct {
	mu       sync.Mutex
	active   bool
	since    time.Time
	lastErr  string
}

var singboxMaintenance = &SingboxMaintenance{}

// Activate stops every singbox child and holds them down.
func (m *SingboxMaintenance) Activate() error {
	m.mu.Lock()
	m.active = true
	m.since = time.Now()
	m.mu.Unlock()
	svc := &SingboxService{}
	if err := svc.StopServices(); err != nil {
		return err
	}
	logger.Info("sing-box: maintenance mode ON — all singbox inbounds stopped and held down")
	return nil
}

// Deactivate lifts the hold and brings every enabled inbound back up.
func (m *SingboxMaintenance) Deactivate() error {
	m.mu.Lock()
	m.active = false
	m.lastErr = ""
	m.mu.Unlock()
	svc := &SingboxService{}
	if err := svc.RestartServices(); err != nil {
		m.mu.Lock()
		m.lastErr = err.Error()
		m.mu.Unlock()
		return err
	}
	logger.Info("sing-box: maintenance mode OFF — inbounds restarted")
	return nil
}

// Blocked reports whether a start attempt should be refused, and why.
func (m *SingboxMaintenance) Blocked() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active, m.lastErr
}

// Status is the maintenance block the core status endpoint merges in.
func (m *SingboxMaintenance) Status() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]any{"active": m.active}
	if m.active {
		out["since"] = m.since.UTC().Format(time.RFC3339)
	}
	return out
}

// baseConfigPayload is the JSON the core-config endpoints return.
type baseConfigPayload struct {
	Config  string `json:"config"`
	Default bool   `json:"isDefault"`
}

// GetSingboxBaseConfig returns the stored (or default) base config text.
func (s *CoreService) GetSingboxBaseConfig() (*baseConfigPayload, error) {
	text, err := baseConfigService.GetBaseConfig()
	if err != nil {
		return nil, err
	}
	payload := &baseConfigPayload{Config: text, Default: text == DefaultSingboxBaseConfig}
	return payload, nil
}

// SaveSingboxBaseConfig validates, stores and applies a new base config.
func (s *CoreService) SaveSingboxBaseConfig(text string) error {
	if err := baseConfigService.SaveBaseConfig(text); err != nil {
		return err
	}
	return baseConfigService.ApplyBaseConfig()
}

// ResetSingboxBaseConfig drops the stored override and re-applies the default.
func (s *CoreService) ResetSingboxBaseConfig() error {
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", singboxBaseSettingKey).Delete(&model.Setting{}).Error; err != nil {
		return err
	}
	return baseConfigService.ApplyBaseConfig()
}

// StartSingboxCore lifts maintenance (if any) and starts every enabled
// singbox inbound. It is the "Start Core" of the maintenance button pair.
func (s *CoreService) StartSingboxCore() error {
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		if err := singboxMaintenance.Deactivate(); err != nil {
			return err
		}
		return nil
	}
	return s.singboxService.RestartServices()
}

// StopSingboxCore enters maintenance: everything stops and stays stopped.
func (s *CoreService) StopSingboxCore() error {
	return singboxMaintenance.Activate()
}

// SingboxMaintenanceStatus exposes maintenance state to the status endpoint.
func (s *CoreService) SingboxMaintenanceStatus() map[string]any {
	return singboxMaintenance.Status()
}
