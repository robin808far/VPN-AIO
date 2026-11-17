#!/usr/bin/env python3
"""Validate GUI-supported XDRIVE services with the pinned executable, offline."""
import json
import pathlib
import re
import subprocess
import sys
import tempfile

binary = str(pathlib.Path(sys.argv[1]).resolve())
pair = subprocess.run([binary, "vlessenc"], check=True, capture_output=True, text=True, timeout=30).stdout
decryption = re.search(r'"decryption":\s*"([^"]+)"', pair).group(1)
encryption = re.search(r'"encryption":\s*"([^"]+)"', pair).group(1)
with tempfile.TemporaryDirectory(prefix="vpn-aio-xdrive-") as tmp:
    for service, extra in [
        ("local", {}),
        ("google-drive", {"secrets": ["client-id", "client-secret", "refresh-token"]}),
        ("s3", {"secrets": ["access-key", "secret-key"],
                "s3": {"endpoint": "https://s3.example.com", "region": "us-east-1",
                       "bucket": "vpn-test", "pathStyle": True}}),
    ]:
        settings = {"service": service, "remoteFolder": "vpn-test", **extra}
        stream = {"network": "xdrive", "security": "none", "xdriveSettings": settings}
        config = {
            "inbounds": [{"listen": "127.0.0.1", "port": 19881, "protocol": "vless",
                          "settings": {"decryption": decryption, "clients": [
                              {"id": "00000000-0000-4000-8000-000000000001"}]},
                          "streamSettings": stream}],
            "outbounds": [{"protocol": "vless", "settings": {"vnext": [{
                "address": "127.0.0.1", "port": 19882, "users": [{
                    "id": "00000000-0000-4000-8000-000000000001", "encryption": encryption}]}]},
                "mux": {"enabled": True, "concurrency": 8, "xudpConcurrency": 8},
                "streamSettings": stream}],
        }
        filename = pathlib.Path(tmp) / "config.json"
        filename.write_text(json.dumps(config))
        result = subprocess.run([binary, "run", "-test", "-c", str(filename)],
                                capture_output=True, text=True, timeout=30)
        assert result.returncode == 0, result.stdout + result.stderr
        print(f"PASS XDRIVE {service} inbound/outbound configuration")
