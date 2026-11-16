#!/usr/bin/env python3
"""Local real-core TCP/UDP smoke tests; no Internet, VM or routing changes.

Usage: python3 test_unit/native-core-smoke.py /absolute/path/to/xray
Requires openssl. Tests AnyTLS UoT, both TUIC UDP modes, Naive h2/h3 and
wrong-password rejection. All processes/listeners/files are temporary.
"""
import contextlib
import http.server
import hashlib
import json
import pathlib
import secrets
import socket
import ssl
import struct
import subprocess
import sys
import tempfile
import threading
import time
import uuid


def port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def receive(s, n):
    data = b""
    while len(data) < n:
        part = s.recv(n - len(data))
        if not part:
            raise ConnectionError("unexpected EOF")
        data += part
    return data


def socks(proxy, target, command=1):
    s = socket.create_connection(("127.0.0.1", proxy), timeout=8)
    try:
        s.sendall(b"\x05\x01\x00")
        assert receive(s, 2) == b"\x05\x00"
        s.sendall(bytes([5, command, 0, 1]) + socket.inet_aton("127.0.0.1") + struct.pack("!H", target))
        header = receive(s, 4)
        assert header[:2] == b"\x05\x00", header
        assert header[3] == 1, header
        host = socket.inet_ntoa(receive(s, 4))
        bound = struct.unpack("!H", receive(s, 2))[0]
        return s, (host, bound)
    except BaseException:
        s.close()
        raise


class HTTP(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        body = b"VPN-AIO-core-smoke-ok"
        self.send_response(200)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass


def tcp_check(proxy, target):
    s, _ = socks(proxy, target)
    with s:
        s.sendall(b"GET / HTTP/1.0\r\nHost: localhost\r\n\r\n")
        data = b""
        while True:
            chunk = s.recv(4096)
            if not chunk:
                break
            data += chunk
            if b"VPN-AIO-core-smoke-ok" in data:
                break
        assert b"VPN-AIO-core-smoke-ok" in data, data


def udp_check(proxy, target):
    control, relay = socks(proxy, 0, 3)
    with control, socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as udp:
        udp.settimeout(8)
        payload = secrets.token_bytes(128)
        frame = b"\x00\x00\x00\x01" + socket.inet_aton("127.0.0.1") + struct.pack("!H", target) + payload
        udp.sendto(frame, relay)
        response = udp.recv(4096)
        assert response[:4] == b"\x00\x00\x00\x01" and response[10:] == payload


def wait_proxy(proc, proxy):
    deadline = time.monotonic() + 10
    while time.monotonic() < deadline:
        if proc.poll() is not None:
            raise RuntimeError("core exited during startup")
        try:
            with socket.create_connection(("127.0.0.1", proxy), timeout=0.2):
                return
        except OSError:
            threading.Event().wait(0.05)
    raise TimeoutError("SOCKS listener did not start")


@contextlib.contextmanager
def core(binary, directory, name, config):
    cfg = directory / (name + ".json")
    cfg.write_text(json.dumps(config))
    log = directory / (name + ".log")
    with log.open("w") as output:
        proc = subprocess.Popen([binary, "run", "-c", str(cfg)], stdout=output, stderr=output)
        try:
            yield proc
        except BaseException:
            print(log.read_text()[-8000:], file=sys.stderr)
            raise
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()


def main():
    binary = str(pathlib.Path(sys.argv[1]).resolve())
    webserver = http.server.ThreadingHTTPServer(("127.0.0.1", 0), HTTP)
    threading.Thread(target=webserver.serve_forever, daemon=True).start()
    udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    udp.bind(("127.0.0.1", 0))
    stop = threading.Event()

    def echo():
        udp.settimeout(0.2)
        while not stop.is_set():
            try:
                data, peer = udp.recvfrom(4096)
                udp.sendto(data, peer)
            except socket.timeout:
                pass

    worker = threading.Thread(target=echo, daemon=True)
    worker.start()
    try:
        with tempfile.TemporaryDirectory(prefix="vpn-aio-core-") as tmp:
            directory = pathlib.Path(tmp)
            crt, key = directory / "cert.pem", directory / "key.pem"
            subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
                            "-subj", "/CN=localhost", "-addext", "subjectAltName=DNS:localhost",
                            "-keyout", str(key), "-out", str(crt)], check=True, capture_output=True)
            cert_pin = hashlib.sha256(ssl.PEM_cert_to_DER_cert(crt.read_text())).hexdigest()
            for protocol, mode in [("anytls", "tcp"), ("tuic", "native"), ("tuic", "quic"), ("naive", "tcp"), ("naive", "udp")]:
                server_port, proxy, api_port = port(), port(), port()
                password, email, uid = secrets.token_hex(16), "smoke@vpn-aio", str(uuid.uuid4())
                account = {"email": email, "password": password, "id": uid}
                network = "tcp" if protocol == "anytls" else protocol
                tls = {"certificates": [{"certificateFile": str(crt), "keyFile": str(key)}]}
                stream = {"network": network, "security": "tls", "tlsSettings": tls}
                if protocol == "naive":
                    stream["naiveSettings"] = {"network": "tcp,udp"}
                server = {"log": {"loglevel": "warning"}, "inbounds": [{"listen": "127.0.0.1", "port": server_port,
                          "protocol": protocol, "settings": {"clients": [account]}, "streamSettings": stream}],
                          "outbounds": [{"protocol": "freedom"}],
                          "api": {"tag": "api", "listen": f"127.0.0.1:{api_port}", "services": ["StatsService"]},
                          "stats": {}, "policy": {"levels": {"0": {"statsUserUplink": True, "statsUserDownlink": True}}}}
                client_stream = {"network": "tcp" if protocol == "anytls" or (protocol == "naive" and mode == "tcp") else protocol,
                                 "security": "tls", "tlsSettings": {"serverName": "localhost", "pinnedPeerCertSha256": cert_pin,
                                 "alpn": ["h2"] if protocol == "naive" and mode == "tcp" else ["h3"]}}
                settings = {**account, "address": "127.0.0.1", "port": server_port}
                if protocol == "tuic":
                    settings["udpRelayMode"] = mode
                if protocol == "naive":
                    settings["network"] = mode
                client = {"log": {"loglevel": "warning"}, "inbounds": [{"listen": "127.0.0.1", "port": proxy,
                          "protocol": "socks", "settings": {"auth": "noauth", "udp": True}}],
                          "outbounds": [{"protocol": protocol, "settings": settings, "streamSettings": client_stream}]}
                with core(binary, directory, "server", server):
                    with core(binary, directory, "client", client) as proc:
                        wait_proxy(proc, proxy)
                        tcp_check(proxy, webserver.server_address[1])
                        if protocol != "naive":
                            udp_check(proxy, udp.getsockname()[1])
                        result = subprocess.run([binary, "api", "statsquery", f"--server=127.0.0.1:{api_port}",
                                                 f"-pattern=user>>>{email}>>>traffic"], check=True, capture_output=True, text=True)
                        counters = json.loads(result.stdout).get("stat", [])
                        for direction in ("uplink", "downlink"):
                            assert any(c["name"].endswith(">>>" + direction) and int(c.get("value", 0)) > 0 for c in counters), counters
                    settings["password"] = "wrong-" + password
                    with core(binary, directory, "wrong-client", client) as proc:
                        wait_proxy(proc, proxy)
                        try:
                            tcp_check(proxy, webserver.server_address[1])
                        except (OSError, AssertionError):
                            pass
                        else:
                            raise AssertionError("wrong password was accepted")
                print(f"PASS {protocol}/{mode}: TCP, {'UDP, ' if protocol != 'naive' else ''}per-user traffic stats, wrong-password rejection", flush=True)
    finally:
        stop.set()
        worker.join(timeout=1)
        udp.close()
        webserver.shutdown()
        webserver.server_close()


if __name__ == "__main__":
    main()
