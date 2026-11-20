package sub

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/util/common"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

func (s *SubService) hostRenders(inbounds []*model.Inbound, format, host string) ([]service.HostRender, error) {
	projected := make([]*model.Inbound, 0, len(inbounds))
	for _, ib := range inbounds {
		clone := *ib
		if strings.HasPrefix(clone.Listen, "@") {
			listen, port, stream, err := s.getFallbackMaster(clone.Listen, clone.StreamSettings)
			if err != nil {
				return nil, err
			}
			clone.Listen, clone.Port, clone.StreamSettings = listen, port, stream
		}
		projected = append(projected, &clone)
	}
	var hosts service.HostService
	return hosts.Renders(projected, format, host)
}

func (s *SubService) hostRemark(inbound *model.Inbound, email string) string {
	h := s.currentHost
	stats, _, _ := s.resolveTraffic(inbound, email)
	left := "∞"
	if stats.Total > 0 {
		left = common.FormatTraffic(max(0, stats.Total-stats.Up-stats.Down))
	}
	days := "∞"
	if stats.ExpiryTime > 0 {
		days = strconv.FormatInt(max(0, (stats.ExpiryTime/1000-time.Now().Unix())/86400), 10)
	}
	return strings.NewReplacer("{{INBOUND}}", inbound.Remark, "{{HOST}}", h.Address,
		"{{ADDRESS}}", inbound.Listen,
		"{{EMAIL}}", email, "{{USERNAME}}", email, "{{TRAFFIC_LEFT}}", left,
		"{{TRAFFIC_USED}}", common.FormatTraffic(stats.Up+stats.Down), "{{TRAFFIC_TOTAL}}", common.FormatTraffic(stats.Total),
		"{{DAYS_LEFT}}", days, "{{PORT}}", strconv.Itoa(inbound.Port)).Replace(h.Remark)
}

func (s *SubService) hostTLSValues() map[string]string {
	h := s.currentHost
	out := map[string]string{}
	if h == nil {
		return out
	}
	if h.Sni != "" || h.KeepSniBlank {
		out["sni"] = h.Sni
	}
	if h.Fingerprint != "" {
		out["fp"] = h.Fingerprint
	}
	if len(h.Alpn) > 0 {
		out["alpn"] = strings.Join(h.Alpn, ",")
	}
	if h.CipherSuites != "" {
		out["cipherSuites"] = h.CipherSuites
	}
	if h.EchConfigList != "" {
		out["ech"] = h.EchConfigList
	}
	if len(h.PinnedPeerCertSha256) > 0 {
		out["pcs"] = strings.Join(h.PinnedPeerCertSha256, ",")
	}
	if h.VerifyPeerCertByName != "" {
		out["verifyPeerCertByName"] = h.VerifyPeerCertByName
	}
	if h.AllowInsecure {
		out["allowInsecure"] = "1"
	}
	return out
}

func (s *SubService) decorateHostLinks(text string, protocol model.Protocol) string {
	switch protocol {
	case model.VMESS, model.VLESS, model.Trojan, model.Shadowsocks, model.Hysteria, model.Hysteria2, model.ANYTLS, model.TUIC, model.NAIVE:
	default:
		return text
	}
	h := s.currentHost
	if h == nil || text == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "vmess://") {
			data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "vmess://"))
			if err != nil {
				continue
			}
			var obj map[string]any
			if json.Unmarshal(data, &obj) != nil {
				continue
			}
			if obj["tls"] != "none" && obj["tls"] != "" {
				for k, v := range s.hostTLSValues() {
					obj[k] = v
				}
			}
			if h.ServerDescription != "" {
				obj["serverDescription"] = h.ServerDescription
			}
			data, _ = json.Marshal(obj)
			lines[i] = "vmess://" + base64.StdEncoding.EncodeToString(data)
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			continue
		}
		if u.Scheme == "vless" || u.Scheme == "trojan" || u.Scheme == "hysteria" || u.Scheme == "hysteria2" || u.Scheme == "hy2" || u.Scheme == "anytls" || u.Scheme == "tuic" {
			q := u.Query()
			if h.Security != "none" && q.Get("security") != "none" {
				for k, v := range s.hostTLSValues() {
					q.Set(k, v)
				}
			}
			if h.MihomoX25519 {
				q.Del("support-x25519mlkem768")
			}
			if h.ServerDescription != "" && s.includeHostDescription {
				u.Fragment += "?serverDescription=" + base64.StdEncoding.EncodeToString([]byte(h.ServerDescription))
			}
			u.RawQuery = q.Encode()
			lines[i] = u.String()
		}
	}
	return strings.Join(lines, "\n")
}

func (s *SubService) applyHostClientStream(stream map[string]any) {
	h := s.currentHost
	if h == nil {
		return
	}
	key := "tlsSettings"
	if stream["security"] == "reality" {
		key = "realitySettings"
	}
	if stream["security"] == "tls" || stream["security"] == "reality" {
		tls, _ := stream[key].(map[string]any)
		if tls == nil {
			tls = map[string]any{}
			stream[key] = tls
		}
		if h.Sni != "" || h.KeepSniBlank {
			tls["serverName"] = h.Sni
		}
		if h.Fingerprint != "" {
			tls["fingerprint"] = h.Fingerprint
		}
		if len(h.Alpn) > 0 {
			tls["alpn"] = h.Alpn
		}
		if h.CipherSuites != "" {
			tls["cipherSuites"] = h.CipherSuites
		}
		if h.EchConfigList != "" {
			tls["echConfigList"] = h.EchConfigList
		}
		if len(h.PinnedPeerCertSha256) > 0 {
			tls["pinnedPeerCertSha256"] = strings.Join(h.PinnedPeerCertSha256, ",")
		}
		if h.VerifyPeerCertByName != "" {
			tls["verifyPeerCertByName"] = h.VerifyPeerCertByName
		}
		// The pinned Xray core removed allowInsecure; RAW/Clash retain their legacy switch.
		delete(tls, "allowInsecure")
	}
	if h.SockoptParams != "" {
		var sockopt map[string]any
		if json.Unmarshal([]byte(h.SockoptParams), &sockopt) == nil {
			stream["sockopt"] = sockopt
		}
	}
}

func (s *SubService) applyHostClash(proxy map[string]any) {
	h := s.currentHost
	if h == nil {
		return
	}
	if h.MihomoIpVersion != "" {
		proxy["ip-version"] = h.MihomoIpVersion
	}
	if h.AllowInsecure {
		proxy["skip-cert-verify"] = true
	}
	if h.Sni != "" || h.KeepSniBlank {
		if _, ok := proxy["sni"]; ok {
			proxy["sni"] = h.Sni
		}
		if _, ok := proxy["servername"]; ok {
			proxy["servername"] = h.Sni
		}
	}
	if h.Fingerprint != "" {
		proxy["client-fingerprint"] = h.Fingerprint
	}
	if len(h.Alpn) > 0 {
		proxy["alpn"] = h.Alpn
	}
	if h.MihomoX25519 {
		if opts, ok := proxy["reality-opts"].(map[string]any); ok {
			opts["support-x25519mlkem768"] = false
		}
	}
}
