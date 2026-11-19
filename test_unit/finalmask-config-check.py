#!/usr/bin/env python3
"""Validate each pinned FinalMask variant through Xray's real config parser."""
import json
import pathlib
import subprocess
import sys
import tempfile


binary = str(pathlib.Path(sys.argv[1]).resolve())
with tempfile.TemporaryDirectory(prefix="vpn-aio-finalmask-") as tmp:
    root = pathlib.Path(tmp)
    subprocess.run(
        ["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
         "-subj", "/CN=localhost", "-keyout", str(root / "key.pem"), "-out", str(root / "cert.pem")],
        check=True, capture_output=True,
    )
    noise_script = root / "noise.lua"
    noise_script.write_text('return "noise"')
    tcp_masks = [
        {"type": "fragment", "settings": {"packets": "tlshello", "lengths": ["100-200"], "delays": ["1-2"]}},
        {"type": "header-custom", "settings": {"clients": [[{"type": "array", "rand": 3, "capture": "client"}]], "servers": [], "errors": [[
            {"type": "str", "packet": "HTTP/1.1 400 Bad Request\\r\\n\\r\\n", "delay": 0}]]}},
        {"type": "sudoku", "settings": {"customTables": []}},
        {"type": "xmc", "settings": {"hostname": "mc.example", "password": "mask-password", "profiles": [{
            "username": "TestUser", "uuid": "00112233-4455-6677-8899-aabbccddeeff",
            "texturesValue": "textures-value", "texturesSignature": "textures-signature"}]}},
        {"type": "filter", "settings": {"allowedSourceIPs": ["10.0.0.0/8"], "blockedSourceIPs": ["203.0.113.7"]}},
    ]
    udp_masks = [
        {"type": "filter", "settings": {"allowedSourceIPs": ["10.0.0.0/8"], "blockedSourceIPs": ["203.0.113.7"]}},
        {"type": "mkcp-legacy", "settings": {"header": "", "value": ""}},
        {"type": "noise", "settings": {"reset": "0", "noise": []}},
        {"type": "noise", "settings": {"script": str(noise_script)}},
        {"type": "salamander", "settings": {"password": "mask-password", "packetSize": "512-1200"}},
        {"type": "sudoku", "settings": {"customTables": []}},
        {"type": "xdns", "settings": {"domains": [{"names": ["example.test"], "types": [16], "edns0": 1232}],
            "resolvers": [{"addrs": ["udp://1.1.1.1:53"], "worker": 32}], "extraPoll": 0}},
        {"type": "xicmp", "settings": {"dgram": True, "ips": ["192.0.2.1"]}},
        {"type": "realm", "settings": {"url": "realm://token@127.0.0.1:443/id", "stunServers": ["127.0.0.1:3478"],
            "portMapping": {"enabled": False}}},
        {"type": "header-custom", "settings": {"mode": "prefix", "client": [{"type": "str", "packet": "head"}], "server": []}},
        {"type": "header-custom", "settings": {"mode": "standalone", "client": [], "server": []}},
    ]
    configs = {
        "tcp": {"inbounds": [{"listen": "127.0.0.1", "port": 19443, "protocol": "vless",
            "settings": {"clients": [], "decryption": "none"}, "streamSettings": {"network": "tcp", "security": "none",
                "finalmask": {"tcp": tcp_masks}}}], "outbounds": [{"protocol": "freedom"}]},
        "udp": {"inbounds": [{"listen": "127.0.0.1", "port": 19444, "protocol": "hysteria",
            "settings": {"version": 2, "clients": []}, "streamSettings": {"network": "hysteria", "security": "tls",
                "hysteriaSettings": {"version": 2}, "tlsSettings": {"alpn": ["h3"], "certificates": [{
                    "certificateFile": str(root / "cert.pem"), "keyFile": str(root / "key.pem")}]},
                "finalmask": {"udp": udp_masks, "quicParams": {"congestion": "bbr", "bbrProfile": "standard"}}}}],
            "outbounds": [{"protocol": "freedom"}]},
        "udphop": {"inbounds": [], "outbounds": [{"protocol": "hysteria", "settings": {
            "version": 2, "address": "proxy.example", "port": 443, "auth": "test"},
            "streamSettings": {"network": "hysteria", "security": "tls",
                "tlsSettings": {"serverName": "proxy.example", "alpn": ["h3"]},
                "hysteriaSettings": {"version": 2},
                "finalmask": {"udp": [{"type": "udphop", "settings": {
                    "mode": "intervalRemote", "interval": "5-10", "remotePorts": "20000-30000"}}]}}}]},
    }
    for name, config in configs.items():
        path = root / f"{name}.json"
        path.write_text(json.dumps(config))
        result = subprocess.run([binary, "run", "-test", "-c", str(path)], capture_output=True, text=True, timeout=30)
        if result.returncode:
            raise SystemExit(result.stdout + result.stderr)
        print(f"PASS FinalMask {name} variants", flush=True)
