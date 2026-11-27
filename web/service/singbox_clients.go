package service

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/xray"
	"gorm.io/gorm"
)

// The client API retains its common clients payload. Only the projection writes
// native users into a sing-box inbound; quota metadata never reaches the core.
func (s *InboundService) saveSingboxClients(data, inbound *model.Inbound, clientID string, adding bool) (bool, error) {
	entries, ok := parseSettingsClients(data.Settings)
	if !ok || len(entries) == 0 || (!adding && len(entries) != 1) {
		return false, fmt.Errorf("expected a nonempty clients array (exactly one for an edit)")
	}
	clients, err := s.GetClients(data)
	if err != nil {
		return false, err
	}
	if err := validateClientIdentities(inbound.Protocol, clients); err != nil {
		return false, err
	}
	if err := validateClientLimits(clients); err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry[singboxRawUserKey] != nil {
			return false, fmt.Errorf("use the common clients payload to edit managed account policy")
		}
		if err := validateSingboxAccount(newAccountFromEntry(entry)); err != nil {
			return false, err
		}
	}
	oldEmail := ""
	if !adding {
		old, err := s.GetClients(inbound)
		if err != nil {
			return false, err
		}
		for _, client := range old {
			if clientIdentity(inbound.Protocol, client) == clientID || client.Email == clientID {
				if oldEmail != "" {
					return false, fmt.Errorf("ambiguous client ID")
				}
				oldEmail = client.Email
			}
		}
		if oldEmail == "" {
			return false, fmt.Errorf("client not found")
		}
	}
	if adding || !sameEmail(oldEmail, stringValue(entries[0]["email"])) {
		if duplicate, err := s.checkEmailsExistForClients(clients); err != nil {
			return false, err
		} else if duplicate != "" {
			return false, duplicateEmailError(duplicate, s.emailHolders(duplicate)...)
		}
	}
	base, err := baseConfigService.GetBaseConfig()
	if err != nil {
		return false, err
	}
	var touched []int
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		accounts := &AccountService{}
		if err := accounts.SyncInboundAccounts(tx, inbound.Id); err != nil {
			return err
		}
		if !adding && oldEmail != stringValue(entries[0]["email"]) {
			newEmail := stringValue(entries[0]["email"])
			changed, err := renameInSettings(tx, accountKey(oldEmail), newEmail)
			if err != nil {
				return err
			}
			touched = append(touched, changed...)
			if err := tx.Model(&model.Account{}).Where("LOWER(TRIM(email)) = ?", accountKey(oldEmail)).Update("email", newEmail).Error; err != nil {
				return err
			}
			if err := tx.Model(&xray.ClientTraffic{}).Where("LOWER(TRIM(email)) = ?", accountKey(oldEmail)).Update("email", newEmail).Error; err != nil {
				return err
			}
			if err := s.UpdateClientIPs(tx, oldEmail, newEmail); err != nil {
				return err
			}
		}
		for _, entry := range entries {
			account, err := accounts.upsertAccountFromEntry(tx, entry, inbound)
			if err != nil {
				return err
			}
			if err := validateSingboxAccount(account); err != nil {
				return err
			}
			if err := ensureSingboxTrafficRow(tx, account, inbound); err != nil {
				return err
			}
			membership := model.AccountInbound{AccountId: account.Id, InboundId: inbound.Id}
			if !adding {
				if err := tx.Where("account_id = ? AND inbound_id = ?", account.Id, inbound.Id).First(&membership).Error; err != nil {
					return err
				}
			} else if err := tx.Create(&membership).Error; err != nil {
				return err
			}
			// Preserve unmodelled fields while allowing this membership's SS key
			// and VLESS flow to be edited through the account form.
			var raw map[string]any
			if membership.Extra != "" {
				if err := json.Unmarshal([]byte(membership.Extra), &raw); err != nil {
					return err
				}
			}
			if raw == nil {
				raw = map[string]any{}
			}
			if inbound.Protocol == model.Shadowsocks && stringValue(entry["password"]) != "" {
				raw["password"] = entry["password"]
			}
			if flow, present := entry["flow"]; present && inbound.Protocol == model.VLESS {
				raw["flow"] = flow
				membership.Flow = stringValue(flow)
			}
			blob, err := json.Marshal(raw)
			if err != nil {
				return err
			}
			membership.Extra = string(blob)
			if err := tx.Save(&membership).Error; err != nil {
				return err
			}
			// Project from Extra so an explicit membership key rotation wins over
			// the old native user, while retaining every other native field.
			var current model.Inbound
			if err := tx.First(&current, inbound.Id).Error; err != nil {
				return err
			}
			current.Settings, _, err = removeAccountFromInbound(&current, account.Email)
			if err != nil {
				return err
			}
			current.Settings, err = projectAccountOntoInbound(&current, account, &membership)
			if err != nil {
				return err
			}
			if err := tx.Model(&model.Inbound{}).Where("id = ?", current.Id).Update("settings", current.Settings).Error; err != nil {
				return err
			}
			changed, err := accounts.ProjectAccount(tx, account.Id)
			if err != nil {
				return err
			}
			touched = append(touched, changed...)
		}
		touched = append(touched, inbound.Id)
		for _, id := range touched {
			var candidate model.Inbound
			if err := tx.First(&candidate, id).Error; err != nil {
				return err
			}
			if !isSingboxInbound(&candidate) {
				continue
			}
			if err := validateSingboxLimits(&candidate); err != nil {
				return err
			}
			candidateSet, err := singboxInboundsWithCandidate(&candidate)
			if err != nil {
				return err
			}
			candidateGroup, err := singboxGroupForCandidate(&candidate, candidateSet)
			if err != nil {
				return err
			}
			runtimeMembers, err := singboxService.runtimeGroupMembers(candidateGroup)
			if err != nil {
				return err
			}
			content, err := singboxService.renderConfigWithBase(candidateGroup.Root, runtimeMembers, base)
			if err != nil {
				return err
			}
			if err := singboxService.checkConfigText(content); err != nil {
				return err
			}
			if err := accounts.SyncInboundAccounts(tx, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	needXray := false
	seen := map[int]bool{}
	for _, id := range touched {
		if seen[id] {
			continue
		}
		seen[id] = true
		changed, err := s.GetInbound(id)
		if err != nil {
			return false, err
		}
		if isSingboxInbound(changed) {
			if err := singboxService.RefreshInbound(id); err != nil {
				return false, fmt.Errorf("account saved; could not apply sing-box inbound %d: %w", id, err)
			}
		} else {
			needXray = true
		}
	}
	return needXray, nil
}

func validateSingboxAccount(account *model.Account) error {
	if account.UserLimitOverride != nil && *account.UserLimitOverride > 0 {
		return fmt.Errorf("account %q: sing-box cannot count physical devices; use the concurrent stream limit instead", account.Email)
	}
	if account.SpeedLimitDown != nil && *account.SpeedLimitDown < 0 {
		return fmt.Errorf("account %q has a negative download speed limit", account.Email)
	}
	if account.SpeedLimitUp != nil && *account.SpeedLimitUp < 0 {
		return fmt.Errorf("account %q has a negative upload speed limit", account.Email)
	}
	if account.StreamLimit < 0 {
		return fmt.Errorf("account %q has a negative concurrent stream limit", account.Email)
	}
	if account.LimitIP < 0 {
		return fmt.Errorf("account %q has a negative IP limit", account.Email)
	}
	return nil
}

func (s *InboundService) updateSingboxAccount(inbound *model.Inbound, email string, update func(*model.Account)) (bool, bool, error) {
	if !isSingboxInbound(inbound) {
		return false, false, nil
	}
	if update == nil {
		return false, true, fmt.Errorf("missing sing-box account update")
	}
	entries, ok := parseSettingsClients(inbound.Settings)
	if !ok {
		return false, true, fmt.Errorf("sing-box inbound %d has invalid users", inbound.Id)
	}
	key := accountKey(email)
	found := false
	for _, entry := range entries {
		if accountKey(AccountEmailFromEntry(entry)) == key {
			found = true
			break
		}
	}
	if !found {
		return false, true, fmt.Errorf("sing-box account %q not found on inbound %d", email, inbound.Id)
	}

	db := database.GetDB()
	accounts := &AccountService{}
	var accountEmail string
	var runtimePolicyChanged bool
	var changedInbounds []int
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := accounts.SyncInboundAccounts(tx, inbound.Id); err != nil {
			return err
		}
		account, err := accounts.GetAccountByEmailTx(tx, email)
		if err != nil {
			return err
		}
		if account == nil {
			return fmt.Errorf("sing-box account %q not found", email)
		}
		oldEnable, oldTotal, oldExpiry, oldReset, oldLimitIP :=
			account.Enable, account.TotalGB, account.ExpiryTime, account.Reset, account.LimitIP
		update(account)
		if err := validateSingboxAccount(account); err != nil {
			return err
		}
		runtimePolicyChanged = oldEnable != account.Enable || oldTotal != account.TotalGB ||
			oldExpiry != account.ExpiryTime || oldReset != account.Reset || oldLimitIP != account.LimitIP
		if err := tx.Save(account).Error; err != nil {
			return err
		}
		changed, err := accounts.ProjectAccount(tx, account.Id)
		if err != nil {
			return err
		}
		changedInbounds = append(changedInbounds, changed...)
		for _, id := range changed {
			if err := accounts.SyncInboundAccounts(tx, id); err != nil {
				return err
			}
		}
		accountEmail = account.Email
		return nil
	})
	if err != nil {
		return false, true, err
	}
	needXrayRestart := false
	if runtimePolicyChanged {
		seen := map[int]bool{}
		for _, id := range changedInbounds {
			if seen[id] {
				continue
			}
			seen[id] = true
			changedInbound, err := s.GetInbound(id)
			if err != nil {
				return false, true, err
			}
			if !isSingboxInbound(changedInbound) {
				needXrayRestart = true
				break
			}
		}
	}
	if err := singboxService.RefreshAccount(accountEmail); err != nil {
		return needXrayRestart, true, fmt.Errorf("account updated; could not apply sing-box account %q: %w", accountEmail, err)
	}
	return needXrayRestart, true, nil
}

func (s *SingboxService) RefreshAccount(email string) error {
	return s.RefreshAccounts([]string{email})
}

func (s *SingboxService) RefreshAccounts(emails []string) error {
	wanted := make(map[string]bool, len(emails))
	for _, email := range emails {
		if key := accountKey(email); key != "" {
			wanted[key] = true
		}
	}
	if len(wanted) == 0 {
		return nil
	}
	inbounds, err := s.GetSingboxInbounds()
	if err != nil {
		return err
	}
	for _, inbound := range inbounds {
		entries, ok := parseSettingsClients(inbound.Settings)
		if !ok {
			continue
		}
		for _, entry := range entries {
			if !wanted[accountKey(AccountEmailFromEntry(entry))] {
				continue
			}
			if err := s.RefreshInbound(inbound.Id); err != nil {
				return fmt.Errorf("refresh sing-box inbound %d: %w", inbound.Id, err)
			}
			break
		}
	}
	// Account policy for native sing-box users is stored in accounts, not in the
	// native users array. Refresh the shared sidecar immediately rather than
	// waiting for traffic or the next one-second invalidation tick.
	MarkSpeedLimitsDirty()
	WriteSpeedLimitsIfDirty()
	return nil
}

func (s *InboundService) deleteSingboxClient(inbound *model.Inbound, email string) (bool, error) {
	settings, removed, err := removeAccountFromInbound(inbound, email)
	if err != nil {
		return false, err
	}
	if !removed {
		return false, fmt.Errorf("client not found")
	}
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("settings", settings).Error; err != nil {
			return err
		}
		if err := (&AccountService{}).SyncInboundAccounts(tx, inbound.Id); err != nil {
			return err
		}
		served, err := s.emailServedOutside(tx, inbound.Id, email)
		if err != nil || served {
			return err
		}
		if err := tx.Where("LOWER(TRIM(email)) = ?", accountKey(email)).Delete(&xray.ClientTraffic{}).Error; err != nil {
			return err
		}
		return s.DelClientIPs(tx, email)
	})
	if err != nil {
		return false, err
	}
	return false, singboxService.RefreshInbound(inbound.Id)
}
