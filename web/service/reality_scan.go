package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// REALITY target probing, in the style of 3x-ui's target scanner: dial a TLS
// endpoint, read back what the handshake negotiated (version, ALPN, key share,
// certificate, chain size, latency) and judge feasibility — REALITY needs a
// target that speaks TLS 1.3 over X25519, which is exactly what a forced
// X25519 handshake proves here.

const (
	realityScanTimeout = 5 * time.Second
	// Upper bound on hosts probed from one CIDR, so a /8 typed by mistake
	// cannot turn the panel into a scanner fleet.
	realityMaxScanHosts = 256
	// Concurrent probes during a Find Targets run.
	realityScanWorkers = 16
)

// Common destinations REALITY is known to work behind, probed when the
// operator leaves the scanner box empty (mirrors 3x-ui's candidate list).
var realityCandidateTargets = []string{
	"www.cloudflare.com",
	"dl.google.com",
	"www.google.com",
	"www.youtube.com",
	"www.amazon.com",
	"aws.amazon.com",
	"www.microsoft.com",
	"www.apple.com",
	"www.samsung.com",
	"www.sony.com",
	"www.nvidia.com",
	"www.amd.com",
	"www.intel.com",
	"www.aliexpress.com",
	"www.tesla.com",
	"www.visa.com",
	"www.paypal.com",
	"www.linkedin.com",
	"www.reddit.com",
	"www.spotify.com",
}

// RealityScanResult is one probed destination. It is the payload of both the
// single-target scan (Scan button) and each row of the target finder.
type RealityScanResult struct {
	Target      string   `json:"target"`
	Feasible    bool     `json:"feasible"`
	SNI         string   `json:"sni"`
	TLSVersion  string   `json:"tlsVersion"`
	ALPN        string   `json:"alpn"`
	KeyExchange string   `json:"keyExchange"`
	Certificate string   `json:"certificate"`
	Issuer      string   `json:"issuer"`
	CertExpires string   `json:"certExpires"`
	CertChain   int      `json:"certChain"`
	LatencyMs   int64    `json:"latencyMs"`
	ServerNames []string `json:"serverNames"`
	Error       string   `json:"error"`
}

// normalizeRealityTarget splits "host", "host:port" or a URL-ish form into a
// dialable host and port. A bare host means 443, like the examples in the form.
func normalizeRealityTarget(raw string) (host, port string, err error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	host0, _, hasPath := strings.Cut(raw, "/")
	if hasPath {
		raw = host0
	}
	if raw == "" {
		return "", "", fmt.Errorf("target is empty")
	}
	if h, p, splitErr := net.SplitHostPort(raw); splitErr == nil {
		pNum, pErr := strconv.Atoi(p)
		if pErr != nil || pNum < 1 || pNum > 65535 {
			return "", "", fmt.Errorf("invalid port %q", p)
		}
		return h, p, nil
	}
	if strings.Contains(raw, ":") {
		// Not host:port (SplitHostPort would have taken it) — an IPv6 literal.
		return strings.Trim(raw, "[]"), "443", nil
	}
	return raw, "443", nil
}

// probeRealityTarget performs one handshake. The X25519 preference is forced
// so a successful TLS 1.3 handshake certifies the exact key exchange REALITY
// requires; servers that refuse X25519 fail the probe and are reported
// infeasible with the handshake error attached.
func probeRealityTarget(host, port, sni string) *RealityScanResult {
	res := &RealityScanResult{Target: net.JoinHostPort(host, port), SNI: sni}
	addr := net.JoinHostPort(host, port)

	dial := func(curves []tls.CurveID) (*tls.Conn, error) {
		dialer := &net.Dialer{Timeout: realityScanTimeout}
		cfg := &tls.Config{
			ServerName:         sni,
			InsecureSkipVerify: true, // we inspect the presented chain, not its trust
			NextProtos:         []string{"h2", "http/1.1"},
			MinVersion:         tls.VersionTLS12,
		}
		if curves != nil {
			cfg.CurvePreferences = curves
		}
		start := time.Now()
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, cfg)
		if err != nil {
			return nil, err
		}
		res.LatencyMs = time.Since(start).Milliseconds()
		return conn, nil
	}

	conn, err := dial([]tls.CurveID{tls.X25519})
	if err != nil {
		// Retry without a curve preference so a server that only negotiates
		// other groups still gets its chain reported, just not "Feasible".
		conn, err = dial(nil)
		if err != nil {
			res.Error = err.Error()
			return res
		}
	} else {
		res.KeyExchange = "X25519"
	}
	defer conn.Close()

	state := conn.ConnectionState()
	res.TLSVersion = tlsVersionName(state.Version)
	res.ALPN = state.NegotiatedProtocol
	if len(state.PeerCertificates) == 0 {
		res.Error = "no certificate presented"
		return res
	}
	certs := state.PeerCertificates
	leaf := certs[0]
	res.Certificate = leaf.Subject.CommonName
	if org := leaf.Subject.Organization; len(org) > 0 {
		res.Certificate = strings.TrimSpace(res.Certificate + " (" + org[0] + ")")
	}
	res.Issuer = leaf.Issuer.CommonName
	res.CertExpires = leaf.NotAfter.UTC().Format("2006-01-02 15:04")
	for _, c := range certs {
		res.CertChain += len(c.Raw)
	}
	res.ServerNames = append(res.ServerNames, leaf.DNSNames...)

	// REALITY feasibility: TLS 1.3 with the forced-X25519 handshake proven
	// above — the pair a REALITY steal-yourself target needs.
	res.Feasible = state.Version == tls.VersionTLS13 && res.KeyExchange != ""
	if res.Feasible && res.SNI == "" && len(res.ServerNames) > 0 {
		res.SNI = res.ServerNames[0]
	}
	return res
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "1.0"
	case tls.VersionTLS11:
		return "1.1"
	case tls.VersionTLS12:
		return "1.2"
	case tls.VersionTLS13:
		return "1.3"
	}
	return ""
}

// ScanRealityTarget validates one operator-typed destination for the Target
// field, its SNI defaulting to the host itself.
func (s *ServerService) ScanRealityTarget(target string) (*RealityScanResult, error) {
	host, port, err := normalizeRealityTarget(target)
	if err != nil {
		return nil, err
	}
	sni := host
	if net.ParseIP(host) != nil {
		sni = "" // certificates for IPs come back via SANs, not SNI
	}
	return probeRealityTarget(host, port, sni), nil
}

// FindRealityTargets probes many destinations at once: an empty query walks
// the common-candidate list, a domain is tried bare and with www., and an IP
// or CIDR is swept on 443 with the certificates' own names as the SNI hint.
func (s *ServerService) FindRealityTargets(query string) ([]*RealityScanResult, error) {
	type job struct {
		host, port, sni string
	}

	var jobs []job
	query = strings.TrimSpace(query)
	switch {
	case query == "":
		for _, t := range realityCandidateTargets {
			jobs = append(jobs, job{host: t, port: "443", sni: t})
		}
	case strings.Contains(query, "/"):
		_, ipnet, err := net.ParseCIDR(query)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q", query)
		}
		bits, total := ipnet.Mask.Size()
		if total-bits > 8 {
			return nil, fmt.Errorf("range /%d too large, scan at most %d hosts (use a /%d or smaller)",
				bits, realityMaxScanHosts, total-8)
		}
		for ip, count := ipnet.IP.Mask(ipnet.Mask), 0; ip != nil && count < realityMaxScanHosts; incIP(ip) {
			host := ip.String()
			jobs = append(jobs, job{host: host, port: "443", sni: ""})
			count++
			if ip.Equal(lastIP(ipnet)) {
				break
			}
		}
	default:
		if ip := net.ParseIP(query); ip != nil {
			jobs = append(jobs, job{host: ip.String(), port: "443", sni: ""})
			break
		}
		host, port, err := normalizeRealityTarget(query)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job{host: host, port: port, sni: host})
		if !strings.HasPrefix(host, "www.") && net.ParseIP(host) == nil {
			jobs = append(jobs, job{host: "www." + host, port: port, sni: "www." + host})
		}
	}

	results := make([]*RealityScanResult, len(jobs))
	sem := make(chan struct{}, realityScanWorkers)
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = probeRealityTarget(j.host, j.port, j.sni)
		}(i, j)
	}
	wg.Wait()

	sort.Slice(results, func(a, b int) bool {
		if results[a].Feasible != results[b].Feasible {
			return results[a].Feasible
		}
		return results[a].LatencyMs < results[b].LatencyMs
	})
	return results, nil
}

// incIP steps an IPv4/IPv6 address in place.
func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			return
		}
	}
}

// lastIP returns the broadcast address of a masked network.
func lastIP(ipnet *net.IPNet) net.IP {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP.Mask(ipnet.Mask))
	for i := range ip {
		ip[i] |= ^ipnet.Mask[i]
	}
	return ip
}
