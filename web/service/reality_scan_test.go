package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNormalizeRealityTarget(t *testing.T) {
	cases := []struct {
		in, host, port string
		wantErr        bool
	}{
		{in: "example.com", host: "example.com", port: "443"},
		{in: "example.com:8443", host: "example.com", port: "8443"},
		{in: " https://example.com:443/ ", host: "example.com", port: "443"},
		{in: "2001:db8::1", host: "2001:db8::1", port: "443"},
		{in: "example.com:0", wantErr: true},
		{in: "example.com:99999", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, c := range cases {
		host, port, err := normalizeRealityTarget(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeRealityTarget(%q) = %q:%q, want error", c.in, host, port)
			}
			continue
		}
		if err != nil || host != c.host || port != c.port {
			t.Errorf("normalizeRealityTarget(%q) = %q:%q, %v; want %q:%q",
				c.in, host, port, err, c.host, c.port)
		}
	}
}

func TestFindRealityTargetsRejectsHugeCIDR(t *testing.T) {
	// A /0 sweep would probe 4 billion hosts: the cap must turn it away
	// before any probe starts, not just cap the loop.
	if _, err := (&ServerService{}).FindRealityTargets("0.0.0.0/0"); err == nil {
		t.Fatal("FindRealityTargets(0.0.0.0/0) = no error, want range-too-large")
	}
}

// A local TLS 1.3 + h2 endpoint: the probe must mark it Feasible, read the
// leaf's CN and SANs, and time the handshake.
func TestProbeRealityTargetFeasible(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "target.local"},
		DNSNames:     []string{"target.local", "alt.target.local"},
		NotAfter:     time.Now().Add(time.Hour),
		NotBefore:    time.Now().Add(-time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			srv := tls.Server(c, &tls.Config{
				Certificates: []tls.Certificate{cert},
				NextProtos:   []string{"h2", "http/1.1"},
			})
			go srv.Handshake()
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	res := probeRealityTarget(host, port, "target.local")
	if !res.Feasible {
		t.Fatalf("probe of TLS 1.3 + X25519 endpoint not Feasible: %+v", res)
	}
	if res.KeyExchange != "X25519" || res.TLSVersion != "1.3" || res.ALPN != "h2" {
		t.Errorf("unexpected negotiation: %+v", res)
	}
	if res.Certificate != "target.local" || !strings.Contains(strings.Join(res.ServerNames, ","), "alt.target.local") {
		t.Errorf("unexpected certificate data: %+v", res)
	}
	if res.CertChain != len(der) {
		t.Errorf("CertChain = %d, want %d", res.CertChain, len(der))
	}
}
