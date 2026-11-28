package service

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/xray"
	"gorm.io/gorm"
)

const singboxRawUserKey = "__vpnAioSingboxUser"

// SingboxEntriesForSettings exposes the same normalized user shape used by the
// accounts mirror to controller code that has to identify a posted user.
func SingboxEntriesForSettings(root map[string]any) ([]map[string]any, bool) {
	return singboxUsersFromRoot(root)
}

func AccountEmailFromEntry(entry map[string]any) string {
	return stringValue(entry["email"])
}

func singboxUsersFromRoot(root map[string]any) ([]map[string]any, bool) {
	users, ok := root["users"].([]any)
	if !ok {
		return nil, false
	}
	protocol := model.Protocol(stringValue(root["type"]))
	out := make([]map[string]any, 0, len(users))
	for _, item := range users {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{singboxRawUserKey: cloneStringMap(raw)}
		if name := stringValue(raw["name"]); name != "" {
			entry["email"] = name
		}
		switch protocol {
		case model.VLESS, model.VMESS:
			entry["id"] = stringValue(raw["uuid"])
			if protocol == model.VLESS {
				entry["flow"] = stringValue(raw["flow"])
			}
		case model.Trojan, model.ANYTLS:
			entry["password"] = stringValue(raw["password"])
		case model.Shadowsocks:
			entry["password"] = stringValue(raw["password"])
			entry["method"] = stringValue(raw["method"])
		case model.TUIC:
			entry["id"] = stringValue(raw["uuid"])
			entry["password"] = stringValue(raw["password"])
		case model.Hysteria2:
			entry["auth"] = stringValue(raw["password"])
		case "shadowtls":
			entry["password"] = stringValue(raw["password"])
		case model.Snell:
			entry["password"] = stringValue(raw["userkey"])
		}
		out = append(out, entry)
	}
	return out, true
}

func validateSingboxUsers(protocol model.Protocol, users []map[string]any) error {
	seen := map[string]bool{}
	for i, user := range users {
		name := stringValue(user["name"])
		if err := ValidateClientEmail(name); err != nil {
			return fmt.Errorf("sing-box user %d: %w", i+1, err)
		}
		key := accountKey(name)
		if seen[key] {
			return fmt.Errorf("duplicate sing-box user name %q", name)
		}
		seen[key] = true
		switch protocol {
		case model.VLESS, model.VMESS:
			if stringValue(user["uuid"]) == "" {
				return fmt.Errorf("sing-box user %q requires a UUID", name)
			}
		case model.Trojan, model.ANYTLS, model.Shadowsocks, model.TUIC, model.Hysteria2, model.Snell, "shadowtls":
			if stringValue(user["password"]) == "" && protocol != model.TUIC {
				return fmt.Errorf("sing-box user %q requires a password", name)
			}
			if protocol == model.TUIC && (stringValue(user["uuid"]) == "" || stringValue(user["password"]) == "") {
				return fmt.Errorf("sing-box TUIC user %q requires a UUID and password", name)
			}
		default:
			return fmt.Errorf("sing-box protocol %q does not support managed users", protocol)
		}
	}
	return nil
}

func allocateSingboxAPIPort() (int, error) {
	db := database.GetDB()
	for range 16 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, err
		}
		port := listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()
		var count int64
		if err := db.Model(&model.Inbound{}).Where("singbox_api_port = ?", port).Count(&count).Error; err != nil {
			return 0, err
		}
		if count == 0 {
			return port, nil
		}
	}
	return 0, fmt.Errorf("could not allocate a unique sing-box stats API port")
}

func singboxRawUser(entry map[string]any) map[string]any {
	if raw, ok := entry[singboxRawUserKey].(map[string]any); ok {
		return cloneStringMap(raw)
	}
	return nil
}

func membershipEntryForStorage(entry map[string]any, inbound *model.Inbound) map[string]any {
	if !isSingboxInbound(inbound) {
		return entry
	}
	if raw := singboxRawUser(entry); raw != nil {
		return raw
	}
	return entry
}

func (s *AccountService) upsertSingboxAccountFromEntry(tx *gorm.DB, entry map[string]any, inbound *model.Inbound) (*model.Account, error) {
	email := stringValue(entry["email"])
	account, err := s.GetAccountByEmailTx(tx, email)
	if err != nil {
		return nil, err
	}
	if account == nil {
		fresh := newAccountFromEntry(entry)
		fresh.Enable = true
		overwriteAccountCredential(fresh, entry, inbound.Protocol)
		if err := tx.Create(fresh).Error; err != nil {
			return nil, err
		}
		return fresh, nil
	}

	updated := *account
	if inbound.Protocol == model.Shadowsocks && shadowsocksKeySize(inbound) > 0 {
		entry = cloneStringMap(entry)
		delete(entry, "password")
	}
	overwriteAccountCredential(&updated, entry, inbound.Protocol)
	if err := tx.Save(&updated).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

func ensureSingboxTrafficRow(tx *gorm.DB, account *model.Account, inbound *model.Inbound) error {
	var traffic xray.ClientTraffic
	err := tx.Where("LOWER(TRIM(email)) = ?", accountKey(account.Email)).First(&traffic).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	traffic = xray.ClientTraffic{
		InboundId:  inbound.Id,
		Email:      account.Email,
		Enable:     account.Enable,
		Total:      account.TotalGB,
		ExpiryTime: account.ExpiryTime,
		Reset:      account.Reset,
	}
	return tx.Create(&traffic).Error
}

func syncSingboxTrafficPolicy(tx *gorm.DB, account *model.Account) error {
	var traffic xray.ClientTraffic
	if err := tx.Where("LOWER(TRIM(email)) = ?", accountKey(account.Email)).First(&traffic).Error; err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	enable := account.Enable && (account.TotalGB <= 0 || traffic.Up+traffic.Down < account.TotalGB)
	if account.ExpiryTime > 0 && account.ExpiryTime <= now && account.Reset == 0 {
		enable = false
	}
	return tx.Model(&xray.ClientTraffic{}).Where("id = ?", traffic.Id).Updates(map[string]any{
		"total":       account.TotalGB,
		"expiry_time": account.ExpiryTime,
		"reset":       account.Reset,
		"enable":      enable,
	}).Error
}

func singboxEntryForRawUser(protocol model.Protocol, raw map[string]any) map[string]any {
	root, _ := json.Marshal(map[string]any{"type": string(protocol), "users": []any{raw}})
	var parsed map[string]any
	_ = json.Unmarshal(root, &parsed)
	entries, _ := singboxUsersFromRoot(parsed)
	if len(entries) == 0 {
		return map[string]any{}
	}
	return entries[0]
}

func renderSingboxUser(account *model.Account, membership *model.AccountInbound, inbound *model.Inbound, existing map[string]any) (map[string]any, error) {
	raw := singboxRawUser(existing)
	if raw == nil && membership != nil && membership.Extra != "" {
		if err := json.Unmarshal([]byte(membership.Extra), &raw); err != nil {
			return nil, fmt.Errorf("inbound %d account %q has invalid stored sing-box user: %w", inbound.Id, account.Email, err)
		}
	}
	if raw == nil {
		raw = map[string]any{}
	}
	raw["name"] = account.Email
	switch inbound.Protocol {
	case model.VLESS, model.VMESS:
		raw["uuid"] = account.UUID
	case model.Trojan, model.ANYTLS:
		raw["password"] = account.Password
	case model.Shadowsocks:
		applyAccountCredential(raw, account, inbound)
	case model.TUIC:
		raw["uuid"] = account.UUID
		raw["password"] = account.Password
	case model.Hysteria2:
		raw["password"] = account.Auth
	case "shadowtls":
		raw["password"] = account.Password
	case model.Snell:
		raw["userkey"] = account.Password
	default:
		return nil, fmt.Errorf("sing-box protocol %q does not support managed users", inbound.Protocol)
	}
	if inbound.Protocol == model.VLESS && membership != nil && membership.Flow != "" {
		raw["flow"] = membership.Flow
	}
	return raw, nil
}

func projectAccountOntoSingboxInbound(inbound *model.Inbound, account *model.Account, membership *model.AccountInbound) (string, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(inbound.Settings), &root); err != nil || root == nil {
		return "", fmt.Errorf("inbound %d has unparseable sing-box settings", inbound.Id)
	}
	users, ok := root["users"].([]any)
	if !ok {
		return "", fmt.Errorf("inbound %d has no sing-box users array", inbound.Id)
	}
	key := accountKey(account.Email)
	found := false
	for i, item := range users {
		raw, ok := item.(map[string]any)
		if !ok || accountKey(stringValue(raw["name"])) != key {
			continue
		}
		user, err := renderSingboxUser(account, membership, inbound, singboxEntryForRawUser(inbound.Protocol, raw))
		if err != nil {
			return "", err
		}
		users[i], found = user, true
		break
	}
	if !found {
		user, err := renderSingboxUser(account, membership, inbound, nil)
		if err != nil {
			return "", err
		}
		users = append(users, user)
	}
	root["users"] = users
	encoded, err := json.MarshalIndent(root, "", "  ")
	return string(encoded), err
}

func cloneStringMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
