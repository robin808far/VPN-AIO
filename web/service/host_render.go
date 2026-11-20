package service

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
)

type HostRender struct {
	Inbound  *model.Inbound
	Host     *model.Host
	bindings map[int]int
}

func hostBindingHere(ids []int, inboundID int, bindings map[int]int) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if binding, ok := bindings[id]; !ok || binding == inboundID {
			return true
		}
	}
	return false
}

func (r HostRender) Allows(client model.Client) bool {
	if !hostBindingHere(client.HostRuleIds, r.Inbound.Id, r.bindings) {
		return true
	}
	return r.Host != nil && slices.Contains(client.HostRuleIds, r.Host.Id)
}

func hostBindings() (map[int]int, error) {
	result := map[int]int{}
	var live []struct{ Id, InboundId int }
	if err := database.GetDB().Model(&model.Host{}).Select("id", "inbound_id").Find(&live).Error; err != nil {
		return nil, err
	}
	var deleted []model.HostTombstone
	if err := database.GetDB().Find(&deleted).Error; err != nil {
		return nil, err
	}
	for _, h := range deleted {
		result[h.HostId] = h.InboundId
	}
	for _, h := range live {
		result[h.Id] = h.InboundId
	}
	return result, nil
}

// AttachMetadata keeps synchronous panel link/QR generation aligned with subscription policy.
func (s *HostService) AttachMetadata(inbounds []*model.Inbound) error {
	if len(inbounds) == 0 {
		return nil
	}
	ids := []int{}
	for _, ib := range inbounds {
		ids = append(ids, ib.Id)
		ib.Hosts = []*model.Host{}
		ib.HostRules = map[string][]int{}
	}
	var rows []*model.Host
	if err := database.GetDB().Where("inbound_id IN ?", ids).Order("sort_order ASC, group_id ASC, id ASC").Find(&rows).Error; err != nil {
		return err
	}
	byInbound := map[int][]*model.Host{}
	for _, h := range rows {
		byInbound[h.InboundId] = append(byInbound[h.InboundId], h)
	}
	bindings, err := hostBindings()
	if err != nil {
		return err
	}
	for _, ib := range inbounds {
		ib.Hosts = byInbound[ib.Id]
		var settings struct {
			Clients []model.Client `json:"clients"`
		}
		if err := json.Unmarshal([]byte(ib.Settings), &settings); err != nil {
			continue
		}
		for _, client := range settings.Clients {
			if hostBindingHere(client.HostRuleIds, ib.Id, bindings) {
				ib.HostRules[client.Email] = slices.Clone(client.HostRuleIds)
			}
		}
	}
	return nil
}

func (s *HostService) Renders(inbounds []*model.Inbound, format, fallback string) ([]HostRender, error) {
	if len(inbounds) == 0 {
		return nil, nil
	}
	if err := s.AttachMetadata(inbounds); err != nil {
		return nil, err
	}
	bindings, err := hostBindings()
	if err != nil {
		return nil, err
	}
	result := []HostRender{}
	sharedSNI := map[string]string{}
	for _, inbound := range inbounds {
		start := len(result)
		for _, row := range inbound.Hosts {
			if row.IsDisabled || slices.Contains(row.ExcludeFromSubTypes, format) {
				continue
			}
			addresses := []string{row.Address}
			if prefix, err := netip.ParsePrefix(row.Address); err == nil {
				addresses, err = randomHostIPs(prefix, row.RandomCount)
				if err != nil {
					return nil, err
				}
			}
			for i, address := range addresses {
				h := *row
				h.Address = address
				if len(addresses) > 1 {
					h.Remark = fmt.Sprintf("%s - %d", row.Remark, i+1)
				}
				if len(h.Snis) > 0 {
					if h.SniSharedRandom {
						if sharedSNI[h.GroupId] == "" {
							sharedSNI[h.GroupId], err = randomHostSNI(h.Snis)
						}
						h.Sni = sharedSNI[h.GroupId]
					} else {
						h.Sni, err = randomHostSNI(h.Snis)
					}
					if err != nil {
						return nil, err
					}
				}
				projected, err := ProjectHost(inbound, &h, fallback)
				if err != nil {
					return nil, err
				}
				if h.OverrideSniFromAddress {
					h.Sni = projected.Listen
				}
				if h.KeepSniBlank {
					h.Sni = ""
				}
				result = append(result, HostRender{Inbound: projected, Host: &h, bindings: bindings})
				if len(result) > 4096 {
					return nil, fmt.Errorf("host expansion exceeds 4096 endpoints")
				}
			}
		}
		if len(result) == start {
			result = append(result, HostRender{Inbound: inbound, bindings: bindings})
		}
	}
	var setting model.Setting
	if err := database.GetDB().Where("key = ?", "subHostSortMode").First(&setting).Error; err == nil && setting.Value == "hosts-first" {
		sort.SliceStable(result, func(i, j int) bool {
			a, b := result[i].Host, result[j].Host
			if a == nil || b == nil {
				return a != nil && b == nil
			}
			if a.SortOrder != b.SortOrder {
				return a.SortOrder < b.SortOrder
			}
			if a.GroupId != b.GroupId {
				return a.GroupId < b.GroupId
			}
			return false
		})
	}
	return result, nil
}

func randomHostSNI(pool []string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	if err != nil {
		return "", err
	}
	return pool[n.Int64()], nil
}

func randomHostIPs(prefix netip.Prefix, count int) ([]string, error) {
	prefix = prefix.Masked()
	count = min(max(count, 1), 100)
	base := prefix.Addr().AsSlice()
	bits := len(base)*8 - prefix.Bits()
	capacity := new(big.Int).Lsh(big.NewInt(1), uint(bits))
	start := int64(0)
	if bits > 1 {
		start = 1
		capacity.Sub(capacity, big.NewInt(1))
		if prefix.Addr().Is4() {
			capacity.Sub(capacity, big.NewInt(1))
		}
	}
	if capacity.IsInt64() && int64(count) > capacity.Int64() {
		count = int(capacity.Int64())
	}
	baseInt := new(big.Int).SetBytes(base)
	seen := map[string]bool{}
	result := []string{}
	for attempts := 0; len(result) < count && attempts < count*100; attempts++ {
		offset, err := rand.Int(rand.Reader, capacity)
		if err != nil {
			return nil, err
		}
		offset.Add(offset, big.NewInt(start))
		offset.Add(offset, baseInt)
		bytes := offset.FillBytes(make([]byte, len(base)))
		address, ok := netip.AddrFromSlice(bytes)
		if !ok {
			return nil, fmt.Errorf("invalid CIDR address")
		}
		text := address.String()
		if !seen[text] {
			seen[text] = true
			result = append(result, text)
		}
	}
	if len(result) != count {
		return nil, fmt.Errorf("could not select distinct CIDR addresses")
	}
	return result, nil
}

func hostObject(raw string) map[string]any {
	obj := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &obj)
	if obj == nil {
		obj = map[string]any{}
	}
	return obj
}

func mergeHostObject(base, override map[string]any) map[string]any {
	for key, value := range override {
		if next, ok := value.(map[string]any); ok {
			prior, _ := base[key].(map[string]any)
			if prior == nil {
				prior = map[string]any{}
			}
			base[key] = mergeHostObject(prior, next)
		} else {
			base[key] = value
		}
	}
	return base
}

// ProjectHost changes a detached export view. Runtime listeners and stored credentials stay intact.
func ProjectHost(inbound *model.Inbound, h *model.Host, fallback string) (*model.Inbound, error) {
	clone := *inbound
	clone.HostProjection = true
	address := h.Address
	if address == "" {
		address = inbound.Listen
		if address == "" || address == "@" || strings.HasPrefix(address, "@") || (net.ParseIP(address) != nil && net.ParseIP(address).IsUnspecified()) {
			address = fallback
		}
	}
	port := h.Port
	if port == 0 {
		port = inbound.Port
	}
	clone.Listen, clone.Port = address, port
	stream := hostObject(inbound.StreamSettings)
	security, _ := stream["security"].(string)
	if h.Security == "tls" || h.Security == "none" {
		security = h.Security
		stream["security"] = security
	}
	if h.Security == "reality" && security != "reality" {
		return nil, fmt.Errorf("Reality hosts require a Reality inbound")
	}
	if security == "none" {
		delete(stream, "tlsSettings")
		delete(stream, "realitySettings")
	}
	if security == "tls" {
		delete(stream, "realitySettings")
	}
	if security == "reality" {
		delete(stream, "tlsSettings")
	}
	sni := h.Sni
	if h.OverrideSniFromAddress {
		sni = address
	}
	if h.KeepSniBlank {
		sni = ""
	}
	key := "tlsSettings"
	if security == "reality" {
		key = "realitySettings"
	}
	if security == "tls" || security == "reality" {
		tls, _ := stream[key].(map[string]any)
		if tls == nil {
			tls = map[string]any{}
			stream[key] = tls
		}
		client, _ := tls["settings"].(map[string]any)
		if client == nil {
			client = map[string]any{}
			tls["settings"] = client
		}
		if sni != "" || h.KeepSniBlank {
			tls["serverName"] = sni
			if security == "reality" {
				tls["serverNames"] = []string{sni}
			}
		}
		if h.Fingerprint != "" {
			client["fingerprint"] = h.Fingerprint
			tls["fingerprint"] = h.Fingerprint
		}
		if len(h.Alpn) > 0 {
			tls["alpn"] = h.Alpn
		}
		if h.CipherSuites != "" {
			tls["cipherSuites"] = h.CipherSuites
		}
		if len(h.PinnedPeerCertSha256) > 0 {
			tls["pinnedPeerCertSha256"] = strings.Join(h.PinnedPeerCertSha256, ",")
		}
		if h.VerifyPeerCertByName != "" {
			tls["verifyPeerCertByName"] = h.VerifyPeerCertByName
		}
		if h.EchConfigList != "" {
			tls["echConfigList"] = h.EchConfigList
			client["echConfigList"] = h.EchConfigList
		}
	}
	for _, key := range []string{"wsSettings", "httpupgradeSettings", "xhttpSettings"} {
		if transport, ok := stream[key].(map[string]any); ok {
			if h.HostHeader != "" {
				transport["host"] = h.HostHeader
			}
			if h.Path != "" {
				transport["path"] = h.Path
			}
			if key == "xhttpSettings" && h.XhttpExtraParams != "" {
				mergeHostObject(transport, hostObject(h.XhttpExtraParams))
			}
		}
	}
	if h.SockoptParams != "" {
		stream["sockopt"] = hostObject(h.SockoptParams)
	}
	if h.FinalMask != "" {
		prior, _ := stream["finalmask"].(map[string]any)
		if prior == nil {
			prior = map[string]any{}
		}
		stream["finalmask"] = mergeHostFinalMask(prior, hostObject(h.FinalMask))
	}
	ep := map[string]any{"dest": address, "port": port, "remark": h.Remark, "forceTls": "same"}
	stream["externalProxy"] = []any{ep}
	settings := hostObject(inbound.Settings)
	if inbound.Protocol == model.VLESS && security == "none" {
		delete(settings, "flow")
	}
	switch inbound.Protocol {
	case model.WGC, model.AWG, model.SSH, model.MTPROTO, model.OPENVPN, model.L2TP, model.PPTP, model.OPENCONNECT, model.SSTP, model.IKEV2:
		settings["externalProxy"] = []any{ep}
	}
	clients, _ := settings["clients"].([]any)
	for _, value := range clients {
		client, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if inbound.Protocol == model.MTPROTO {
			client["externalProxy"] = []any{ep}
		}
		if inbound.Protocol == model.VLESS && security == "none" {
			delete(client, "flow")
		}
		if inbound.Protocol == model.VLESS && h.VlessRoute != "" {
			id, _ := client["id"].(string)
			parsed, err := uuid.Parse(id)
			if err != nil {
				return nil, fmt.Errorf("VLESS route requires a valid UUID")
			}
			route, _ := strconv.Atoi(h.VlessRoute)
			parsed[6], parsed[7] = byte(route>>8), byte(route)
			client["id"] = parsed.String()
		}
	}
	b, err := json.Marshal(stream)
	if err != nil {
		return nil, err
	}
	clone.StreamSettings = string(b)
	b, err = json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	clone.Settings = string(b)
	return &clone, nil
}

// ManagedHostInbounds lets native config exporters use the same binding and endpoint resolution.
func ManagedHostInbounds(inbound *model.Inbound, email, format, fallback string) ([]*model.Inbound, bool, error) {
	if inbound.HostProjection || database.GetDB() == nil || inbound.Id == 0 {
		return nil, false, nil
	}
	svc := HostService{}
	renders, err := svc.Renders([]*model.Inbound{inbound}, format, fallback)
	if err != nil {
		return nil, false, err
	}
	var clients []model.Client
	var inboundSvc InboundService
	clients, err = inboundSvc.GetClients(inbound)
	if err != nil {
		return nil, false, err
	}
	client := model.Client{Email: email}
	for _, candidate := range clients {
		if candidate.Email == email {
			client = candidate
			break
		}
	}
	managed := false
	result := []*model.Inbound{}
	for _, render := range renders {
		if render.Host != nil || !render.Allows(client) {
			managed = true
		}
		if render.Allows(client) {
			result = append(result, render.Inbound)
		}
	}
	return result, managed, nil
}

func mergeHostFinalMask(base, extra map[string]any) map[string]any {
	for key, value := range extra {
		if key == "tcp" || key == "udp" {
			prior, _ := base[key].([]any)
			next, _ := value.([]any)
			if len(next) > 0 {
				base[key] = append(prior, next...)
			}
			continue
		}
		if key == "quicParams" {
			if _, exists := base[key]; exists {
				continue
			}
		}
		base[key] = value
	}
	return base
}
