package service

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"gorm.io/gorm"
)

type HostService struct{}

func supportsManagedHosts(protocol model.Protocol) bool {
	switch protocol {
	case model.VMESS, model.VLESS, model.Trojan, model.Shadowsocks, model.Hysteria, model.Hysteria2, model.ANYTLS, model.TUIC, model.NAIVE, model.MASQUE,
		model.L2TP, model.PPTP, model.OPENVPN, model.OPENCONNECT, model.SSTP, model.IKEV2, model.WGC, model.AWG, model.WireGuard, model.MTPROTO, model.SSH:
		return true
	}
	return false
}

func (s *HostService) SortMode() (string, error) {
	var settings SettingService
	return settings.getString("subHostSortMode")
}
func (s *HostService) SetSortMode(mode string) error {
	if mode != "inbound-first" && mode != "hosts-first" {
		return fmt.Errorf("invalid host sorting mode")
	}
	var settings SettingService
	return settings.saveSetting("subHostSortMode", mode)
}

type HostGroupView struct {
	model.HostGroup
	Editable bool `json:"editable"`
}

func hostGroupRows(rows []*model.Host) []*model.HostGroup {
	groups := []*model.HostGroup{}
	byID := map[string]*model.HostGroup{}
	for _, h := range rows {
		g := byID[h.GroupId]
		if g == nil {
			g = &model.HostGroup{Host: *h, InboundIds: []int{}, Hosts: []string{}}
			byID[h.GroupId] = g
			groups = append(groups, g)
		}
		if !slices.Contains(g.InboundIds, h.InboundId) {
			g.InboundIds = append(g.InboundIds, h.InboundId)
		}
		addr := h.Address
		if h.Port != 0 {
			addr = net.JoinHostPort(addr, strconv.Itoa(h.Port))
		}
		if !slices.Contains(g.Hosts, addr) {
			g.Hosts = append(g.Hosts, addr)
		}
	}
	return groups
}

func (s *HostService) Raw(user *model.User) ([]*model.Host, error) {
	rows := []*model.Host{}
	if user == nil || user.IsReseller {
		return rows, nil
	}
	q := database.GetDB().Order("sort_order ASC, group_id ASC, id ASC")
	if !user.IsSuperAdmin {
		q = q.Where("inbound_id IN (?)", database.GetDB().Model(&model.InboundAccess{}).Select("inbound_id").Where("user_id = ?", user.Id))
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (s *HostService) List(user *model.User) ([]*HostGroupView, error) {
	rows, err := s.Raw(user)
	if err != nil {
		return nil, err
	}
	result := []*HostGroupView{}
	for _, g := range hostGroupRows(rows) {
		err := s.authorize(database.GetDB(), user, g.GroupId, nil)
		result = append(result, &HostGroupView{HostGroup: *g, Editable: err == nil && user.Can(model.PermEditInbound)})
	}
	return result, nil
}

func (s *HostService) authorize(tx *gorm.DB, user *model.User, groupID string, ids []int) error {
	if user == nil || user.IsReseller {
		return fmt.Errorf("not found")
	}
	if groupID != "" {
		var existing []int
		if err := tx.Model(&model.Host{}).Where("group_id = ?", groupID).Pluck("inbound_id", &existing).Error; err != nil {
			return err
		}
		if len(existing) == 0 {
			return fmt.Errorf("not found")
		}
		ids = append(slices.Clone(ids), existing...)
	}
	ids = uniqueHostIDs(ids)
	var existing int64
	if err := tx.Model(&model.Inbound{}).Where("id IN ?", ids).Count(&existing).Error; err != nil {
		return err
	}
	if int(existing) != len(ids) {
		return fmt.Errorf("not found")
	}
	if !user.IsSuperAdmin && len(ids) > 0 {
		var grants int64
		if err := tx.Model(&model.InboundAccess{}).Where("user_id = ? AND inbound_id IN ?", user.Id, ids).Count(&grants).Error; err != nil {
			return err
		}
		if int(grants) != len(ids) {
			return fmt.Errorf("not found")
		}
	}
	return nil
}

func uniqueHostIDs(ids []int) []int {
	out := []int{}
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

var hostDNSLabel = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func splitHostAddress(value string, defaultPort int) (string, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", defaultPort, nil
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", 0, fmt.Errorf("invalid host address")
	}
	address, port := value, defaultPort
	if host, p, err := net.SplitHostPort(value); err == nil {
		address = host
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 65535 {
			return "", 0, fmt.Errorf("invalid host port")
		}
		port = n
	} else if strings.Count(value, ":") == 1 {
		return "", 0, fmt.Errorf("invalid host:port")
	}
	address = strings.Trim(address, "[]")
	if address == "" {
		return "", port, nil
	}
	if _, err := netip.ParseAddr(address); err == nil {
		return address, port, nil
	}
	if _, err := netip.ParsePrefix(address); err == nil {
		return address, port, nil
	}
	if len(address) > 253 {
		return "", 0, fmt.Errorf("host address is too long")
	}
	for _, label := range strings.Split(strings.TrimSuffix(address, "."), ".") {
		if !hostDNSLabel.MatchString(label) {
			return "", 0, fmt.Errorf("invalid host address %q", address)
		}
	}
	return address, port, nil
}

func validateHostGroup(g *model.HostGroup) error {
	g.Remark = strings.TrimSpace(g.Remark)
	if g.Remark == "" || len(g.Remark) > 256 || strings.IndexFunc(g.Remark, unicode.IsControl) >= 0 {
		return fmt.Errorf("host remark must contain 1–256 printable bytes")
	}
	g.InboundIds = uniqueHostIDs(g.InboundIds)
	if len(g.InboundIds) == 0 || len(g.InboundIds) > 100 || len(g.Hosts) > 100 || max(1, len(g.Hosts))*len(g.InboundIds) > 2000 {
		return fmt.Errorf("invalid host group size")
	}
	if g.Port < 0 || g.Port > 65535 {
		return fmt.Errorf("host port must be between 0 and 65535")
	}
	if g.Security == "" {
		g.Security = "same"
	}
	if !slices.Contains([]string{"same", "tls", "none", "reality"}, g.Security) {
		return fmt.Errorf("invalid host security")
	}
	if len(g.ServerDescription) > 256 || len(g.Snis) > 100 || len(g.Tags) > 10 || len(g.Alpn) > 16 || len(g.PinnedPeerCertSha256) > 16 {
		return fmt.Errorf("host field exceeds its limit")
	}
	for _, entry := range g.PinnedPeerCertSha256 {
		for _, pin := range strings.Split(entry, ",") {
			pin = strings.TrimSpace(pin)
			if pin == "" {
				continue
			}
			data, err := hex.DecodeString(strings.ReplaceAll(pin, ":", ""))
			if err != nil || len(data) != 32 {
				return fmt.Errorf("certificate pins must be SHA-256 hex values")
			}
		}
	}
	if g.RandomCount == 0 {
		g.RandomCount = 1
	}
	if g.RandomCount < 1 || g.RandomCount > 100 {
		return fmt.Errorf("randomCount must be between 1 and 100")
	}
	if g.OverrideSniFromAddress && g.KeepSniBlank {
		return fmt.Errorf("choose address SNI or blank SNI, not both")
	}
	if g.MihomoIpVersion != "" && !slices.Contains([]string{"dual", "ipv4", "ipv6", "ipv4-prefer", "ipv6-prefer"}, g.MihomoIpVersion) {
		return fmt.Errorf("invalid Mihomo IP version")
	}
	for _, format := range g.ExcludeFromSubTypes {
		if !slices.Contains([]string{"raw", "json", "clash"}, format) {
			return fmt.Errorf("invalid subscription format")
		}
	}
	if g.VlessRoute != "" {
		n, err := strconv.Atoi(g.VlessRoute)
		if err != nil || n < 0 || n > 65535 {
			return fmt.Errorf("invalid VLESS route")
		}
	}
	for _, value := range append(append(slices.Clone(g.Snis), g.Sni, g.HostHeader, g.Path, g.Fingerprint, g.VerifyPeerCertByName, g.ServerDescription), g.Tags...) {
		if len(value) > 2048 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return fmt.Errorf("host fields cannot contain control characters")
		}
	}
	for _, raw := range []string{g.MuxParams, g.SockoptParams, g.FinalMask, g.XhttpExtraParams} {
		if raw == "" {
			continue
		}
		var obj map[string]any
		if len(raw) > 65536 || json.Unmarshal([]byte(raw), &obj) != nil || obj == nil {
			return fmt.Errorf("advanced host settings must be JSON objects")
		}
	}
	if len(g.Hosts) == 0 {
		g.Hosts = []string{""}
	}
	for _, addr := range g.Hosts {
		if _, _, err := splitHostAddress(addr, g.Port); err != nil {
			return err
		}
	}
	return nil
}

func buildManagedHosts(groupID string, g *model.HostGroup) []*model.Host {
	rows := []*model.Host{}
	snis := []string{}
	for _, sni := range g.Snis {
		if sni = strings.TrimSpace(sni); sni != "" && !slices.Contains(snis, sni) {
			snis = append(snis, sni)
		}
	}
	if len(snis) == 0 && strings.TrimSpace(g.Sni) != "" {
		snis = []string{strings.TrimSpace(g.Sni)}
	}
	for _, input := range g.Hosts {
		address, port, _ := splitHostAddress(input, g.Port)
		for _, inboundID := range g.InboundIds {
			h := g.Host
			h.Id, h.GroupId, h.InboundId, h.Address, h.Port = 0, groupID, inboundID, address, port
			h.LegacySource, h.CreatedAt, h.UpdatedAt = "", 0, 0
			h.Snis, h.Sni = snis, ""
			if len(snis) > 0 {
				h.Sni = snis[0]
			}
			rows = append(rows, &h)
		}
	}
	return rows
}

func (s *HostService) Save(user *model.User, groupID string, g *model.HostGroup) ([]*model.Host, error) {
	if user == nil || user.IsReseller || !user.Can(model.PermEditInbound) {
		return nil, fmt.Errorf("not found")
	}
	if err := validateHostGroup(g); err != nil {
		return nil, err
	}
	var saved []*model.Host
	err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := s.authorize(tx, user, groupID, g.InboundIds); err != nil {
			return err
		}
		var inbounds []model.Inbound
		if err := tx.Where("id IN ?", g.InboundIds).Find(&inbounds).Error; err != nil {
			return err
		}
		for _, inbound := range inbounds {
			if !supportsManagedHosts(inbound.Protocol) {
				return fmt.Errorf("protocol %s has no supported Host endpoint export", inbound.Protocol)
			}
			if g.Security == "reality" {
				if hostObject(inbound.StreamSettings)["security"] != "reality" {
					return fmt.Errorf("Reality hosts require Reality inbounds")
				}
			}
		}
		var old []*model.Host
		if groupID == "" {
			groupID = uuid.NewString()
			var last int
			if err := tx.Model(&model.Host{}).Select("COALESCE(MAX(sort_order), -1)").Scan(&last).Error; err != nil {
				return err
			}
			g.SortOrder = last + 1
		} else if err := tx.Where("group_id = ?", groupID).Order("id ASC").Find(&old).Error; err != nil {
			return err
		}
		if len(old) > 0 {
			g.SortOrder = old[0].SortOrder
		}
		saved = buildManagedHosts(groupID, g)
		used := map[int]bool{}
		for _, exact := range []bool{true, false} {
			for _, h := range saved {
				if h.Id != 0 {
					continue
				}
				for _, candidate := range old {
					if used[candidate.Id] || candidate.InboundId != h.InboundId || (exact && (candidate.Address != h.Address || candidate.Port != h.Port)) {
						continue
					}
					h.Id, h.CreatedAt = candidate.Id, candidate.CreatedAt
					used[h.Id] = true
					break
				}
			}
		}
		for _, h := range saved {
			if err := tx.Save(h).Error; err != nil {
				return err
			}
		}
		for _, h := range old {
			if !used[h.Id] {
				if err := database.DeleteHosts(tx, "id = ?", h.Id); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return saved, err
}

func formatHostEndpoint(host string, port int) string {
	return net.JoinHostPort(strings.Trim(host, "[]"), strconv.Itoa(port))
}

func (s *HostService) Mutate(user *model.User, ids []string, action string, enable bool) error {
	if user == nil || user.IsReseller || !user.Can(model.PermEditInbound) || len(ids) == 0 || len(ids) > 2000 {
		return fmt.Errorf("not found")
	}
	return database.GetDB().Transaction(func(tx *gorm.DB) error {
		seen := map[string]bool{}
		for _, id := range ids {
			if seen[id] {
				return fmt.Errorf("duplicate host group")
			}
			seen[id] = true
			if err := s.authorize(tx, user, id, nil); err != nil {
				return err
			}
		}
		switch action {
		case "delete":
			return database.DeleteHosts(tx, "group_id IN ?", ids)
		case "enable":
			return tx.Model(&model.Host{}).Where("group_id IN ?", ids).Update("is_disabled", !enable).Error
		case "reorder":
			// Only submitted groups move; foreign groups and their relative positions stay intact.
			var rows []*model.Host
			if err := tx.Where("group_id IN ?", ids).Order("sort_order ASC, group_id ASC, id ASC").Find(&rows).Error; err != nil {
				return err
			}
			positions := []int{}
			for _, group := range hostGroupRows(rows) {
				positions = append(positions, group.SortOrder)
			}
			sort.Ints(positions)
			for i := 1; i < len(positions); i++ {
				if positions[i] <= positions[i-1] {
					positions[i] = positions[i-1] + 1
				}
			}
			for i, id := range ids {
				if err := tx.Model(&model.Host{}).Where("group_id = ?", id).Update("sort_order", positions[i]).Error; err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("invalid host action")
	})
}

func importInboundHosts(tx *gorm.DB, inbound *model.Inbound) error {
	if len(inbound.Hosts) == 0 {
		return database.ImportLegacyHosts(tx, inbound)
	}
	if len(inbound.Hosts) > 2000 {
		return fmt.Errorf("too many imported hosts")
	}
	groups := map[string]string{}
	remap := map[int]int{}
	for _, source := range inbound.Hosts {
		if source == nil {
			return fmt.Errorf("invalid imported host")
		}
		g := &model.HostGroup{Host: *source, InboundIds: []int{inbound.Id}, Hosts: []string{source.Address}}
		blankRemark := g.Remark == ""
		if blankRemark {
			g.Remark = "imported"
		}
		if err := validateHostGroup(g); err != nil {
			return err
		}
		if blankRemark {
			g.Remark = ""
		}
		groupID := groups[source.GroupId]
		if groupID == "" {
			groupID = uuid.NewString()
			groups[source.GroupId] = groupID
		}
		for _, row := range buildManagedHosts(groupID, g) {
			if err := tx.Create(row).Error; err != nil {
				return err
			}
			if source.Id > 0 {
				remap[source.Id] = row.Id
			}
		}
	}
	settings := hostObject(inbound.Settings)
	clients, _ := settings["clients"].([]any)
	for _, value := range clients {
		client, ok := value.(map[string]any)
		if !ok {
			continue
		}
		original, _ := client["hostRuleIds"].([]any)
		if len(original) == 0 {
			continue
		}
		email, _ := client["email"].(string)
		ids := []int{}
		for _, raw := range original {
			n, ok := raw.(float64)
			if !ok {
				return fmt.Errorf("invalid imported Host binding")
			}
			oldID := int(n)
			if id, exists := remap[oldID]; exists {
				ids = append(ids, id)
				continue
			}
			if inbound.HostRules != nil && !slices.Contains(inbound.HostRules[email], oldID) {
				continue
			}
			// Reserve a tombstone so source IDs never bind to unrelated local Hosts.
			stub := &model.Host{GroupId: uuid.NewString(), InboundId: inbound.Id, IsDisabled: true}
			if err := tx.Create(stub).Error; err != nil {
				return err
			}
			if err := database.DeleteHosts(tx, "id = ?", stub.Id); err != nil {
				return err
			}
			ids = append(ids, stub.Id)
		}
		client["hostRuleIds"] = ids
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	inbound.Settings = string(data)
	return tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Update("settings", inbound.Settings).Error
}
