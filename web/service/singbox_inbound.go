package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/xray"
	"gorm.io/gorm"
)

func validateInboundCore(inbound *model.Inbound) error {
	switch inbound.Core {
	case "", "xray":
		inbound.Core = ""
	case "singbox":
	default:
		return fmt.Errorf("unknown inbound core %q", inbound.Core)
	}
	if isSingboxProtocol(inbound.Protocol) {
		inbound.Core = "singbox"
	}
	return nil
}

// sing-box's per-user controls are enforced by the patched bundled core through
// the shared speedlimits.json sidecar. Quota/expiry still remain panel-owned;
// the native users array carries credentials only.
func validateSingboxLimits(inbound *model.Inbound) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(inbound.Settings), &root); err != nil || root == nil {
		return fmt.Errorf("sing-box settings must be a JSON object")
	}
	if _, exists := root["clients"]; exists {
		return fmt.Errorf("sing-box settings must use the native users array, not an Xray clients array")
	}
	var users []map[string]any
	if err := json.Unmarshal(root["users"], &users); err != nil || users == nil {
		return fmt.Errorf("sing-box managed accounts require a users array")
	}
	if inbound.Protocol == "shadowtls" {
		var version int
		if err := json.Unmarshal(root["version"], &version); err != nil || version != 3 {
			return fmt.Errorf("sing-box ShadowTLS managed accounts require version 3")
		}
	}
	return validateSingboxUsers(inbound.Protocol, users)
}

// saveSingboxInbound keeps validation, persistence and runtime reconciliation in
// that order. No runtime reads through a second handle while a row is uncommitted.
func (s *InboundService) saveSingboxInbound(inbound, previous *model.Inbound) (*model.Inbound, bool, error) {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	if previous == nil {
		inbound.Id = 0
	}
	if previous != nil && inbound.Protocol != previous.Protocol {
		return inbound, false, fmt.Errorf("changing a sing-box inbound's protocol is not supported")
	}
	if err := validateSingboxLimits(inbound); err != nil {
		return inbound, false, err
	}
	if previous != nil {
		inbound.SingboxAPIPort = previous.SingboxAPIPort
	}
	if inbound.SingboxAPIPort <= 0 {
		port, err := allocateSingboxAPIPort()
		if err != nil {
			return inbound, false, fmt.Errorf("allocate sing-box stats API port: %w", err)
		}
		inbound.SingboxAPIPort = port
	}
	var listener struct {
		Listen string `json:"listen"`
		Port   int    `json:"listen_port"`
	}
	if err := json.Unmarshal([]byte(inbound.Settings), &listener); err != nil {
		return inbound, false, err
	}
	// The raw listener is authoritative. Persist its actual socket so conflict
	// checks, the list and exported links describe what the core will bind.
	inbound.Listen, inbound.Port = listener.Listen, listener.Port
	if inbound.Port < 1 || inbound.Port > 65535 {
		return inbound, false, fmt.Errorf("sing-box listen_port must be between 1 and 65535")
	}
	if err := s.validateInboundConfig(inbound); err != nil {
		return inbound, false, err
	}
	clients, err := s.GetClients(inbound)
	if err != nil {
		return inbound, false, err
	}
	var known []string
	if previous != nil {
		oldClients, err := s.GetClients(previous)
		if err != nil {
			return inbound, false, err
		}
		for _, client := range oldClients {
			known = append(known, client.Email)
		}
	}
	if duplicate, err := s.checkEmailsExistExcludingKnown(clients, inbound.Id, known); err != nil {
		return inbound, false, err
	} else if duplicate != "" {
		return inbound, false, duplicateEmailError(duplicate, s.emailHolders(duplicate)...)
	}
	inbound.Core = "singbox"
	inbound.StreamSettings, inbound.Sniffing = "", ""
	inbound.ClientStats = nil
	// Persist an already-expired/depleted listener as disabled. The runtime
	// reconciliation also enforces this for existing rows, but doing it on the
	// value being saved keeps the returned object and the committed row aligned.
	if inbound.Enable && ((inbound.Total > 0 && inbound.Up+inbound.Down >= inbound.Total) ||
		(inbound.ExpiryTime > 0 && inbound.ExpiryTime <= time.Now().UnixMilli())) {
		inbound.Enable = false
	}
	inbound.Tag = fmt.Sprintf("inbound-%d", inbound.Port)
	if inbound.Listen != "" && inbound.Listen != "0.0.0.0" && inbound.Listen != "::" && inbound.Listen != "::0" {
		inbound.Tag = fmt.Sprintf("inbound-%s:%d", inbound.Listen, inbound.Port)
	}
	if previous != nil {
		inbound.UserId, inbound.SortOrder = previous.UserId, previous.SortOrder
		inbound.AllTime = previous.AllTime
	}
	if _, targetTag, err := singboxDetourTarget(inbound); err != nil {
		return inbound, false, err
	} else if targetTag == "" {
		// Clearing the raw detour in the editor must also clear the internal
		// stable relation; otherwise a previous target would silently return.
		inbound.SingboxDetourInboundID = 0
	}
	exists, err := s.checkPortExist(inbound.Listen, inbound.Port, inbound.Id)
	if err != nil {
		return inbound, false, err
	}
	if exists {
		return inbound, false, fmt.Errorf("port already exists: %d", inbound.Port)
	}
	if inbound.Enable {
		if err := s.checkPortConflicts(inbound, inbound.Id); err != nil {
			return inbound, false, err
		}
	}
	svc := &SingboxService{}
	content, err := svc.buildConfig(inbound)
	if err != nil {
		return inbound, false, err
	}
	// Turning an existing inbound off must remain possible after core removal.
	if inbound.Enable || svc.Available() {
		if err := svc.checkConfigText(content); err != nil {
			return inbound, false, err
		}
	} else if previous == nil || previous.Enable == inbound.Enable || strings.TrimSpace(inbound.Settings) != strings.TrimSpace(previous.Settings) {
		return inbound, false, fmt.Errorf("install sing-box before saving its configuration")
	}
	snapshot, err := snapshotSingboxAccounts(inbound, previous)
	if err != nil {
		return inbound, false, err
	}
	var previousGroup *singboxProcessGroup
	if previous != nil {
		if groups, groupErr := singboxGroupsFromDB(); groupErr == nil {
			previousGroup = singboxGroupForID(groups, previous.Id)
		} else {
			return inbound, false, groupErr
		}
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		if previous == nil {
			inbound.Id = 0
			if err := tx.Create(inbound).Error; err != nil {
				return err
			}
			if err := importInboundHosts(tx, inbound); err != nil {
				return err
			}
			return (&AccountService{}).SyncInboundAccounts(tx, inbound.Id)
		}
		if err := database.SyncLegacyHosts(tx, previous, inbound); err != nil {
			return err
		}
		if err := tx.Omit("ClientStats").Save(inbound).Error; err != nil {
			return err
		}
		return (&AccountService{}).SyncInboundAccounts(tx, inbound.Id)
	})
	if err != nil {
		return inbound, false, err
	}
	groups, groupErr := singboxGroupsFromDB()
	if groupErr != nil {
		return inbound, false, groupErr
	}
	newGroup := singboxGroupForID(groups, inbound.Id)
	if newGroup == nil {
		return inbound, false, fmt.Errorf("saved sing-box inbound %d is not in a process group", inbound.Id)
	}
	applyErr := error(nil)
	if previousGroup != nil && !sameSingboxGroup(previousGroup, newGroup) {
		applyErr = svc.restartServices()
	} else {
		applyErr = svc.reconcileInbound(inbound)
	}
	if applyErr != nil {
		rollbackErr := database.GetDB().Transaction(func(tx *gorm.DB) error {
			if previous != nil {
				if err := tx.Omit("ClientStats").Save(previous).Error; err != nil {
					return err
				}
			} else {
				if err := database.DeleteHosts(tx, "inbound_id = ?", inbound.Id); err != nil {
					return err
				}
				if err := tx.Delete(&model.Inbound{}, inbound.Id).Error; err != nil {
					return err
				}
			}
			return snapshot.restore(tx, inbound.Id)
		})
		if rollbackErr != nil {
			return inbound, false, errors.Join(fmt.Errorf("sing-box could not apply the inbound: %w", applyErr), fmt.Errorf("could not restore database: %w", rollbackErr))
		}
		if blocked, _ := singboxMaintenance.Blocked(); !blocked {
			_ = svc.restartServices()
		}
		return inbound, false, fmt.Errorf("sing-box could not apply the inbound; the previous configuration was restored: %w", applyErr)
	}
	return inbound, false, nil
}

// Runtime validation can still fail at bind time after a committed save. Keep
// the affected account/membership rows as well as the old inbound so rollback
// cannot leave a new user, a rotated credential, or a deleted quota behind.
type singboxAccountSnapshot struct {
	keys        []string
	accounts    []model.Account
	memberships []model.AccountInbound
	traffic     []xray.ClientTraffic
}

func snapshotSingboxAccounts(inbound, previous *model.Inbound) (*singboxAccountSnapshot, error) {
	snapshot := &singboxAccountSnapshot{}
	seen := map[string]bool{}
	for _, source := range []*model.Inbound{inbound, previous} {
		if source == nil {
			continue
		}
		entries, _ := parseSettingsClients(source.Settings)
		for _, entry := range entries {
			key := accountKey(stringValue(entry["email"]))
			if key != "" && !seen[key] {
				seen[key] = true
				snapshot.keys = append(snapshot.keys, key)
			}
		}
	}
	if len(snapshot.keys) == 0 {
		return snapshot, nil
	}
	db := database.GetDB()
	if err := db.Where("LOWER(TRIM(email)) IN ?", snapshot.keys).Find(&snapshot.accounts).Error; err != nil {
		return nil, err
	}
	if err := db.Where("LOWER(TRIM(email)) IN ?", snapshot.keys).Find(&snapshot.traffic).Error; err != nil {
		return nil, err
	}
	if inbound.Id > 0 {
		if err := db.Where("inbound_id = ?", inbound.Id).Find(&snapshot.memberships).Error; err != nil {
			return nil, err
		}
	}
	return snapshot, nil
}

func (s *singboxAccountSnapshot) restore(tx *gorm.DB, inboundID int) error {
	if len(s.keys) == 0 {
		return nil
	}
	if err := tx.Where("inbound_id = ?", inboundID).Delete(&model.AccountInbound{}).Error; err != nil {
		return err
	}
	oldIDs := map[int]bool{}
	for _, account := range s.accounts {
		oldIDs[account.Id] = true
		if err := tx.Save(&account).Error; err != nil {
			return err
		}
	}
	var current []model.Account
	if err := tx.Where("LOWER(TRIM(email)) IN ?", s.keys).Find(&current).Error; err != nil {
		return err
	}
	for _, account := range current {
		if oldIDs[account.Id] {
			continue
		}
		var served int64
		if err := tx.Model(&model.AccountInbound{}).Where("account_id = ?", account.Id).Count(&served).Error; err != nil {
			return err
		}
		if served > 0 {
			return fmt.Errorf("account %q acquired another membership during rollback", account.Email)
		}
		if err := tx.Delete(&account).Error; err != nil {
			return err
		}
	}
	oldTraffic := map[string]bool{}
	for _, traffic := range s.traffic {
		oldTraffic[accountKey(traffic.Email)] = true
		var count int64
		if err := tx.Model(&xray.ClientTraffic{}).Where("id = ?", traffic.Id).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Create(&traffic).Error; err != nil {
				return err
			}
		}
	}
	for _, key := range s.keys {
		if !oldTraffic[key] {
			if err := tx.Where("LOWER(TRIM(email)) = ?", key).Delete(&xray.ClientTraffic{}).Error; err != nil {
				return err
			}
		}
	}
	if len(s.memberships) > 0 {
		return tx.Create(&s.memberships).Error
	}
	return nil
}
