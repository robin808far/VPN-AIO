#!/usr/bin/env python3
"""Exercise 6.1.0 audit fixes against a disposable local panel container.

Requires fix-admin / Fix-610-Disposable-Only on port 20983 and sing-box installed.
Never runs outside Docker and never accepts a remote panel address.
"""
import http.cookiejar
import http.server
import json
import os
import socket
import struct
import subprocess
import threading
import urllib.parse
import urllib.request
from pathlib import Path

assert Path('/.dockerenv').exists(), 'Disposable Docker fixture only'
BASE = 'http://127.0.0.1:20983'
checks = []
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def call(path, data=None):
    body = urllib.parse.urlencode(data).encode() if data is not None else None
    request = urllib.request.Request(BASE + path, data=body)
    request.add_header('X-Requested-With', 'XMLHttpRequest')
    if body is not None:
        request.add_header('Content-Type', 'application/x-www-form-urlencoded')
    with opener.open(request, timeout=30) as response:
        return json.load(response)


def check(name, passed):
    checks.append({'name': name, 'passed': bool(passed)})
    print(('PASS ' if passed else 'FAIL ') + name, flush=True)


def listening(port):
    try:
        with socket.create_connection(('127.0.0.1', port), timeout=1):
            return True
    except OSError:
        return False


def inbound(ident):
    return call('/panel/api/inbounds/get/' + str(ident))['obj']


assert call('/login', {'username': 'fix-admin', 'password': 'Fix-610-Disposable-Only'})['success']
token = call('/panel/nodes/tokens', {'name': 'fix-http', 'scopes': 'node-sync'})
check('A9 browser form token creation', token.get('success'))
node = call('/panel/nodes/add', {'name': 'original', 'scheme': 'http', 'address': '127.0.0.1',
            'port': 20983, 'token': token.get('msg', ''), 'basePath': '/', 'enabled': 'true'})
node_id = (node.get('obj') or {}).get('id')
check('node self-probe with form-created token', (node.get('obj') or {}).get('status') == 'online')
renamed = call('/panel/nodes/update', {'id': node_id, 'name': 'renamed', 'scheme': 'http',
               'address': '127.0.0.1', 'port': 20983, 'token': '', 'basePath': '/', 'enabled': 'true'})
check('A13 rename persists and empty token is preserved', (renamed.get('obj') or {}).get('name') == 'renamed'
      and (renamed.get('obj') or {}).get('status') == 'online')
call('/panel/nodes/del/' + str(node_id), {})
call('/panel/nodes/tokens/del/' + str((token.get('obj') or {}).get('id')), {})

check('A12 fresh base save', call('/panel/core/singbox/base-config',
      {'config': '{"log":{"level":"warn"},"outbounds":[{"type":"direct"}]}' }).get('success'))
check('A12 null base rejected', not call('/panel/core/singbox/base-config', {'config': 'null'}).get('success'))
status = call('/panel/core/status')['obj']['cores']
check('A17 sing-box reports installed version', next(row for row in status if row['name'] == 'singbox')['version'] == '1.12.14')

uuid = '11111111-1111-4111-8111-111111111111'
block = {'type': 'vless', 'listen': '127.0.0.1', 'listen_port': 34410,
         'users': [{'name': 'fixture-user', 'uuid': uuid}]}
form = {'remark': 'audit-fix-raw', 'core': 'singbox', 'protocol': 'vless',
        'listen': '127.0.0.1', 'port': 34410, 'enable': 'true', 'settings': json.dumps(block)}
check('A2 quota refused before creating unmanaged inbound',
      not call('/panel/api/inbounds/add', {**form, 'total': 1024}).get('success'))
created = call('/panel/api/inbounds/add', form)
check('A4 raw inbound save succeeds', created.get('success'))
ident = (created.get('obj') or {})['id']
check('A4 save starts listener automatically', listening(34410))

# A real VLESS connection to this committed listener through the exact core.
class Origin(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        payload = b'fixture-body-' + b'x' * (128 * 1024)
        self.send_response(200)
        self.send_header('Content-Length', str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, *args):
        pass


origin = http.server.ThreadingHTTPServer(('127.0.0.1', 34411), Origin)
threading.Thread(target=origin.serve_forever, daemon=True).start()
binary = '/opt/VPN-AIO/bin/sing-box'
client_config = {'inbounds': [{'type': 'mixed', 'listen': '127.0.0.1', 'listen_port': 34412}],
                 'outbounds': [{'type': 'vless', 'server': '127.0.0.1', 'server_port': 34410, 'uuid': uuid}]}
path = Path('/tmp/audit-610-client.json')
path.write_text(json.dumps(client_config))
os.chmod(path, 0o600)
client = subprocess.Popen([binary, 'run', '-c', str(path)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
try:
    # The client process is newly exec'd; use its local SOCKS listener as a
    # bounded readiness gate instead of assuming a fixed startup delay.
    import time
    deadline = time.monotonic() + 5
    while not listening(34412) and time.monotonic() < deadline:
        time.sleep(.05)
    with socket.create_connection(('127.0.0.1', 34412), timeout=5) as tunnel:
        tunnel.sendall(b'\x05\x01\x00')
        assert tunnel.recv(2) == b'\x05\x00'
        tunnel.sendall(b'\x05\x01\x00\x01' + socket.inet_aton('127.0.0.1') + struct.pack('!H', 34411))
        reply = tunnel.recv(10)
        assert reply[:2] == b'\x05\x00'
        tunnel.sendall(b'GET / HTTP/1.0\r\nHost: localhost\r\n\r\n')
        response = b''
        while True:
            data = tunnel.recv(65536)
            if not data:
                break
            response += data
    check('real VLESS tunnel through automatically started sing-box', b'fixture-body-' in response and len(response) > 128 * 1024)
finally:
    client.terminate()
    client.wait(timeout=5)
    path.unlink(missing_ok=True)
    origin.shutdown()

block['listen_port'] = 34413
edited = call('/panel/api/inbounds/update/' + str(ident), {'settings': json.dumps(block), 'port': 34413})
check('A4 port edit moves listener after commit', edited.get('success') and listening(34413) and not listening(34410))
before = inbound(ident)
bad = {**block, 'type': 'not-a-protocol'}
refused = call('/panel/api/inbounds/update/' + str(ident), {'settings': json.dumps(bad)})
after = inbound(ident)
check('A5 invalid config refused and prior row/listener retained', not refused.get('success') and after['settings'] == before['settings'] and listening(34413))
check('A10 sing-box to Xray engine change explicitly refused',
      not call('/panel/api/inbounds/update/' + str(ident), {'core': ''}).get('success') and inbound(ident)['core'] == 'singbox')
check('A2 managed client add refused for raw inbound',
      not call('/panel/api/inbounds/addClient', {'id': ident, 'settings': '{"clients":[{"id":"fixture-id","email":"unsupported","enable":true}]}'}).get('success'))
catalog = call('/panel/core/catalog')['obj']['cores']
check('A8 catalog counts shared-protocol sing-box inbound', next(row for row in catalog if row['name'] == 'singbox')['inbounds'] == 1)
refused = call('/panel/core/uninstall', {'cores': 'singbox'})
check('A8 uninstall requires explicit inbound disposition', not refused.get('success') and listening(34413))
check('A4 disable stops only the saved listener', call('/panel/api/inbounds/update/' + str(ident), {'enable': 'false'}).get('success') and not listening(34413))
check('A4 enable starts listener', call('/panel/api/inbounds/update/' + str(ident), {'enable': 'true'}).get('success') and listening(34413))
call('/panel/api/inbounds/del/' + str(ident), {})
check('delete stops sing-box listener', not listening(34413))

summary = {'passed': sum(row['passed'] for row in checks), 'failed': sum(not row['passed'] for row in checks), 'checks': checks}
Path('/tmp/audit-610-fixes.json').write_text(json.dumps(summary, indent=2) + '\n')
print(json.dumps(summary, indent=2))
assert summary['failed'] == 0
