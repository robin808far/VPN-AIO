#!/usr/bin/env python3
"""Happ HTTP contract check using a fresh panel inside a disposable container."""
import base64
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import signal
import subprocess
import time
import urllib.parse
import urllib.request


ROOT = Path("/opt/VPN-AIO")
BINARY = ROOT / "VPN-AIO-amd64"
PANEL_PORT = 20983
SUB_PORT = 20984


def require(condition, name):
    if not condition:
        raise RuntimeError(name)
    print("PASS " + name, flush=True)


def main():
    if not Path("/.dockerenv").exists() or os.environ.get("VPN_AIO_HAPP_CHECK_ACK") != "yes":
        raise SystemExit("Run only in an acknowledged disposable container.")
    os.chdir(ROOT)
    password = secrets.token_urlsafe(24)
    subprocess.run(
        [str(BINARY), "--port", str(PANEL_PORT), "--user", "happ-test-admin", "--pass", password, "--path", "/"],
        check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    cookies = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookies))

    def api(path, data=None):
        body = urllib.parse.urlencode(data).encode() if data is not None else None
        request = urllib.request.Request(f"http://127.0.0.1:{PANEL_PORT}{path}", data=body)
        request.add_header("X-Requested-With", "XMLHttpRequest")
        with opener.open(request, timeout=10) as response:
            return json.load(response)

    def login():
        deadline = time.monotonic() + 60
        while time.monotonic() < deadline and panel.poll() is None:
            try:
                if api("/login", {"username": "happ-test-admin", "password": password}).get("success"):
                    return
            except Exception:
                pass
            time.sleep(0.3)
        raise RuntimeError("fresh panel login failed")

    def restart():
        require(api("/panel/setting/restartPanel", {}).get("success"), "restart request accepted")
        time.sleep(5)
        login()
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                fetch("/sub/happ-fixture", "Happ/2.0 (Android)")
                return
            except Exception:
                time.sleep(0.3)
        raise RuntimeError("subscription listener unavailable after restart")

    def fetch(path, user_agent):
        request = urllib.request.Request(f"http://127.0.0.1:{SUB_PORT}{path}")
        request.add_header("User-Agent", user_agent)
        with urllib.request.urlopen(request, timeout=10) as response:
            return response.headers, response.read()

    with open("/tmp/happ-panel.log", "w") as log:
        panel = subprocess.Popen([str(BINARY)], stdout=log, stderr=log)
        try:
            login()
            require(subprocess.check_output([str(BINARY), "-v"], text=True).strip() == "3.1.0", "version 3.1.0")
            settings = api("/panel/setting/all", {})["obj"]
            require(settings["subHappAutoDetect"] is False, "fresh install does not opt into Happ overrides")
            settings.update({
                "subEnable": True, "subJsonEnable": True, "subClashEnable": True,
                "subListen": "127.0.0.1", "subPort": SUB_PORT, "subHappAutoDetect": True,
                "subHappProviderId": "fixture-provider", "subHappSubInfoText": "Fixture banner",
                "subHappSubInfoColor": "green", "subHappHideSettings": True, "subHappAutoConnect": True,
                "subHappAutoConnectType": "lastused", "subHappTunMode": "gvisor",
                "subHappPerAppMode": "bypass", "subHappPerAppList": "org.example.app",
                "subHappFallbackUrl": "https://backup.example/sub", "subEnableRouting": True,
            })
            require(api("/panel/setting/update", settings).get("success"), "Happ settings saved through authenticated API")
            reread = api("/panel/setting/all", {})["obj"]
            require(reread["subHappSubInfoText"] == "Fixture banner" and reread["subHappAutoConnectType"] == "lastused",
                    "settings read back from database")
            inbound = {
                "enable": True, "listen": "127.0.0.1", "port": 30443, "protocol": "vless",
                "remark": "Happ fixture", "tag": "happ-fixture-inbound", "up": 0, "down": 0, "total": 0,
                "settings": json.dumps({"decryption": "none", "clients": [{
                    "id": "11111111-2222-3333-4444-555555555555", "email": "happ-fixture",
                    "subId": "happ-fixture", "enable": True, "totalGB": 1024**3,
                    "expiryTime": (int(time.time()) + 86400) * 1000,
                }]}),
                "streamSettings": json.dumps({"network": "tcp", "security": "none"}),
                "sniffing": json.dumps({"enabled": False}), "allocate": "",
            }
            require(api("/panel/api/inbounds/add", inbound).get("success"), "isolated subscriber fixture created")
            restart()
            expected = {
                "ProviderID": "fixture-provider", "Sub-Info-Text": "Fixture banner",
                "Sub-Info-Color": "green", "Hide-Settings": "1", "Subscription-Autoconnect": "1",
                "Subscription-Autoconnect-Type": "lastused", "Tun-Mode": "gvisor",
                "Per-App-Proxy-Mode": "bypass", "Per-App-Proxy-List": "org.example.app",
                "Fallback-Url": "https://backup.example/sub",
            }
            for path in ("/sub/happ-fixture", "/json/happ-fixture", "/clash/happ-fixture"):
                headers, body = fetch(path, "Happ/2.0 (Android)")
                for key, value in expected.items():
                    require(headers.get(key) == value, f"{path}: {key}")
                require("total=1073741824" in headers.get("Subscription-Userinfo", ""), f"{path}: quota unchanged")
                if path.startswith("/sub/"):
                    require(b"vless://" in base64.b64decode(body), "raw subscription payload importable")
                elif path.startswith("/json/"):
                    require(isinstance(json.loads(body), (dict, list)), "JSON subscription payload valid")
                else:
                    require(b"proxies:" in body, "Clash subscription payload retained")
                other_headers, other_body = fetch(path, "v2rayNG/1.9")
                require(other_headers.get("ProviderID") is None and other_headers.get("Hide-Settings") is None,
                        f"{path}: no Happ metadata for other clients")
                require(other_body == body, f"{path}: app metadata does not change config payload")
            with opener.open(f"http://127.0.0.1:{PANEL_PORT}/panel/settings", timeout=10) as response:
                html = response.read().decode()
            require("subHappAutoDetect" in html and "settings/panel/subscription/happ" not in html,
                    "authenticated settings page renders embedded Happ controls")
            settings["subHappAutoDetect"] = False
            require(api("/panel/setting/update", settings).get("success"), "Happ management disabled through API")
            restart()
            headers, _ = fetch("/sub/happ-fixture", "Happ/2.0")
            require(headers.get("ProviderID") is None and headers.get("Hide-Settings") is None,
                    "Happ opt-out suppresses metadata after restart")
            require(headers.get("Routing-Enable") == "true", "legacy routing unchanged after Happ opt-out")
            print(json.dumps({
                "version": "3.1.0", "sha256": hashlib.sha256(BINARY.read_bytes()).hexdigest(),
                "coverage": "real panel HTTP with simulated Happ User-Agent; not a real Happ app test",
            }), flush=True)
        finally:
            if panel.poll() is None:
                panel.send_signal(signal.SIGTERM)
                try:
                    panel.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    panel.kill()
                    panel.wait()


if __name__ == "__main__":
    main()
