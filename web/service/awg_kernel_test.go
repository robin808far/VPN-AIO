//go:build linux

package service

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	wgctrl "github.com/Jipok/wgctrl-go"
	"github.com/Jipok/wgctrl-go/wgtypes"
)

// Opt-in integration test for the actual kernel ABI, not a mock. Both UDP sockets
// are born in the host namespace; moving the client link isolates its inner route
// while leaving its encrypted transport reachable over host loopback.
func TestAWGKernelHandshakeAndPeerRemoval(t *testing.T) {
	if os.Getenv("VPN_AIO_KERNEL_TEST") != "1" {
		t.Skip("set VPN_AIO_KERNEL_TEST=1 on a disposable Linux test host with amneziawg loaded")
	}
	if os.Geteuid() != 0 {
		t.Fatal("kernel integration test requires root")
	}
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
			t.Fatalf("ip %v: %v: %s", args, err, out)
		}
	}
	ns := fmt.Sprintf("aio-awg-%d", os.Getpid())
	server := fmt.Sprintf("aios%d", os.Getpid())
	client := fmt.Sprintf("aioc%d", os.Getpid())
	run("netns", "add", ns)
	t.Cleanup(func() { _ = exec.Command("ip", "netns", "del", ns).Run() })
	run("link", "add", server, "type", "amneziawg")
	t.Cleanup(func() { _ = exec.Command("ip", "link", "del", server).Run() })
	run("link", "add", client, "type", "amneziawg")
	t.Cleanup(func() { _ = exec.Command("ip", "link", "del", client).Run() })
	ctl, err := wgctrl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer ctl.Close()
	serverKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	clientKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	protection, _ := wgtypes.GenerateKey()
	obfs := (&awgSettings{H1: "11111111", H2: "22222222", H3: "33333333", H4: "44444444",
		S1: intPtr(16), S2: intPtr(20), S3: intPtr(12), S4: intPtr(12),
		HeaderProtectionKey:    base64.StdEncoding.EncodeToString(protection[:]),
		ContentPaddingAddition: "8-16", RekeyAfterTime: "100-120", RekeyTimeout: "3-6", RejectAfterTime: "160-200",
		KeepaliveTimeout: "8-16", MaxHandshakeAttempts: "15-25", RandomTrailers: awgBoolPtr(true), DisableCookies: awgBoolPtr(true),
	}).resolveObfs()
	scfg := wgtypes.Config{PrivateKey: &serverKey, ReplacePeers: true, Peers: []wgtypes.PeerConfig{{
		PublicKey: clientKey.PublicKey(), ReplaceAllowedIPs: true,
		AllowedIPs: []net.IPNet{{IP: net.ParseIP("198.18.55.2"), Mask: net.CIDRMask(32, 32)}},
	}}}
	obfs.apply(&scfg)
	if err := ctl.ConfigureDevice(server, scfg); err != nil {
		t.Fatal(err)
	}
	run("addr", "add", "198.18.55.1/30", "dev", server)
	run("link", "set", server, "up")
	sdev, err := ctl.Device(server)
	if err != nil {
		t.Fatal(err)
	}
	keepalive := time.Second
	ccfg := wgtypes.Config{PrivateKey: &clientKey, ReplacePeers: true, Peers: []wgtypes.PeerConfig{{
		PublicKey: serverKey.PublicKey(), ReplaceAllowedIPs: true,
		AllowedIPs:                  []net.IPNet{{IP: net.ParseIP("198.18.55.1"), Mask: net.CIDRMask(32, 32)}},
		Endpoint:                    &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: sdev.ListenPort},
		PersistentKeepaliveInterval: &keepalive,
	}}}
	obfs.apply(&ccfg)
	if err := ctl.ConfigureDevice(client, ccfg); err != nil {
		t.Fatal(err)
	}
	run("link", "set", client, "netns", ns)
	run("-n", ns, "link", "set", "lo", "up")
	run("-n", ns, "addr", "add", "198.18.55.2/30", "dev", client)
	run("-n", ns, "link", "set", client, "up")
	if out, err := exec.Command("ip", "netns", "exec", ns, "ping", "-c", "3", "-W", "5", "198.18.55.1").CombinedOutput(); err != nil {
		t.Fatalf("tunnel ping: %v: %s", err, out)
	}
	sdev, err = ctl.Device(server)
	if err != nil || len(sdev.Peers) != 1 {
		t.Fatalf("read kernel peer: %v, device=%+v", err, sdev)
	}
	p := sdev.Peers[0]
	if p.LastHandshakeTime.IsZero() || p.ReceiveBytes == 0 || p.TransmitBytes == 0 {
		t.Fatalf("missing handshake or traffic counters: %+v", p)
	}
	if got:=sdev.S3;got!=12 {t.Errorf("kernel S3=%d, want 12",got)}
	if got:=sdev.S4;got!=12 {t.Errorf("kernel S4=%d, want 12",got)}
	if err := ctl.ConfigureDevice(server, wgtypes.Config{Peers: []wgtypes.PeerConfig{{PublicKey: clientKey.PublicKey(), Remove: true}}}); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("ip", "netns", "exec", ns, "ping", "-c", "1", "-W", "2", "198.18.55.1").Run(); err == nil {
		t.Fatal("traffic still passed after removing the peer")
	}
	t.Logf("kernel handshake, bidirectional counters (%d/%d) and peer removal verified", p.ReceiveBytes, p.TransmitBytes)
}

func awgBoolPtr(v bool) *bool { return &v }
