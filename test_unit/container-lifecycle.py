#!/usr/bin/env python3
"""Run ONLY inside a disposable Docker container, never on a VPN host.

Checks fresh startup, authenticated API, embedded native core protocols and
destructive uninstall. Requires Python 3, openssl, procps, iproute2 and nftables.
This does not validate distro kernels, systemd boot or installed VPN daemons.
"""
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import time
import urllib.parse
import urllib.request


ROOT = Path('/opt/VPN-AIO')
BINARY = ROOT / 'VPN-AIO-amd64'
PORT = 20983
CHECKS = []


def check(name, condition):
    CHECKS.append({'name': name, 'passed': bool(condition)})
    print(('PASS ' if condition else 'FAIL ') + name, flush=True)
    if not condition:
        raise RuntimeError(name)


def run(args, timeout=90, check=True):
    result = subprocess.run(args, capture_output=True, text=True, timeout=timeout)
    if check and result.returncode:
        raise RuntimeError(f'{args[0]} exited {result.returncode}: {result.stdout[-1200:]} {result.stderr[-1200:]}')
    return result


def call(opener, path, data=None):
    body = urllib.parse.urlencode(data).encode() if data is not None else None
    request = urllib.request.Request(f'http://127.0.0.1:{PORT}{path}', data=body)
    request.add_header('X-Requested-With', 'XMLHttpRequest')
    with opener.open(request, timeout=5) as response:
        return json.load(response)


def main():
    if os.geteuid() != 0 or not Path('/.dockerenv').exists():
        raise SystemExit('This destructive check requires root INSIDE a disposable Docker container.')
    if os.environ.get('VPN_AIO_CONTAINER_LIFECYCLE_ACK') != 'yes':
        raise SystemExit('Set VPN_AIO_CONTAINER_LIFECYCLE_ACK=yes only inside an isolated disposable container.')
    # Docker's default /etc/hosts, /etc/hostname and /etc/resolv.conf mounts are
    # harmless. Refuse any bind mount over directories uninstall may remove or
    # alter, even when the caller supplied the acknowledgement by mistake.
    protected = ('/etc', '/opt', '/usr', '/var')
    harmless_etc = {'/etc/hosts', '/etc/hostname', '/etc/resolv.conf'}
    mounts = set()
    for line in Path('/proc/self/mountinfo').read_text().splitlines():
        fields = line.split()
        if len(fields) > 4:
            mount = fields[4].replace('\\040', ' ').replace('\\011', '\t').replace('\\134', '\\')
            if mount not in harmless_etc:
                mounts.add(mount)
    unsafe = sorted(m for m in mounts if any(m == p or m.startswith(p + '/') for p in protected))
    if unsafe:
        raise SystemExit('Refusing destructive container test with protected bind mounts: ' + ', '.join(unsafe))
    os.chdir(ROOT)
    os.environ.setdefault('XRAY_LOCATION_ASSET', str(ROOT / 'bin'))
    password = secrets.token_urlsafe(24)
    check('candidate version', run([str(BINARY), '-v']).stdout.strip() == '2.1.0')
    digest = hashlib.sha256(BINARY.read_bytes()).hexdigest()
    run([str(BINARY), '--port', str(PORT), '--user', 'lifecycle-admin', '--pass', password, '--path', '/'])
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    with open('/tmp/panel.log', 'w') as log:
        panel = subprocess.Popen([str(BINARY)], stdout=log, stderr=log)
        try:
            deadline = time.monotonic() + 60
            while time.monotonic() < deadline and panel.poll() is None:
                try:
                    if call(opener, '/login', {'username': 'lifecycle-admin', 'password': password}).get('success'):
                        break
                except Exception:
                    pass
                time.sleep(.3)
            else:
                raise RuntimeError('Panel login unavailable; see /tmp/panel.log')
            check('authenticated fresh panel', True)
            check('authenticated inbound API', call(opener, '/panel/api/inbounds/list').get('success'))
            check('SQLite database created', (ROOT / 'VPN-AIO.db').is_file())
            deadline = time.monotonic() + 30
            while time.monotonic() < deadline:
                status = run([str(BINARY), 'ctl', 'cores.status'], check=False)
                if any('xray' in line and 'running' in line for line in status.stdout.splitlines()):
                    break
                time.sleep(.3)
            else:
                raise RuntimeError('Embedded Xray did not become ready')
            check('embedded Xray running', True)
            for script in ('native-core-smoke.py', 'xdrive-config-check.py',
                           'finalmask-config-check.py', 'transport-settings-config-check.py'):
                result = run([sys.executable, '/checks/' + script, str(ROOT / 'bin/xray-linux-amd64')], timeout=180)
                Path('/tmp/' + script + '.log').write_text(result.stdout + result.stderr)
                check(script, result.returncode == 0)

            # These are explicit filesystem fixtures, not claims of installed VPN
            # daemons. Test that backups/keys disappear and unowned files survive.
            for directory in ('cert', 'backups'):
                (ROOT / directory).mkdir(exist_ok=True)
                (ROOT / directory / 'fixture-secret').write_text('disposable fixture')
            Path('/usr/libexec/vpn-ui').mkdir(parents=True, exist_ok=True)
            Path('/usr/libexec/vpn-ui/uninstall-fixture').write_text('disposable bundle fixture')
            marker = ROOT / 'operator-notes.txt'
            marker.write_text('retain unrelated operator files\n')
            unit = Path('/etc/systemd/system/VPN-AIO.service')
            unit.parent.mkdir(parents=True, exist_ok=True)
            unit.write_text('[Service]\nExecStart=' + str(BINARY) + '\n')
            menu = Path('/usr/bin/VPN-AIO')
            menu.write_text('#!/bin/sh\n# disposable menu fixture\n')
            host_file = Path('/etc/ipsec.secrets')
            host_file.write_text('# operator-owned sentinel\n')
            # Exercise real namespace-local firewall and policy-route cleanup.
            run(['nft', 'add', 'table', 'ip', 'vpn'], check=False)
            check('firewall cleanup fixture exists', run(['nft', 'list', 'table', 'ip', 'vpn'], check=False).returncode == 0)
            run(['ip', 'rule', 'add', 'fwmark', '1', 'lookup', '100'])
            run(['ip', 'route', 'add', 'local', '0.0.0.0/0', 'dev', 'lo', 'table', '100'])
            result = run([str(BINARY), '--uninstall', '--yes'], timeout=120)
            Path('/tmp/uninstall.log').write_text(result.stdout + result.stderr)
            check('uninstall reported no teardown errors', 'Encountered errors' not in result.stdout)
            panel.wait(timeout=10)
            check('panel process stopped', panel.poll() is not None)
            for path in (BINARY, ROOT / 'VPN-AIO.db', ROOT / 'VPN-AIO.db-wal', ROOT / 'VPN-AIO.db-shm',
                         ROOT / 'VPN-AIO.db-journal', ROOT / 'bin', ROOT / 'cert', ROOT / 'backups',
                         ROOT / 'vpn-ui.sock', Path('/usr/libexec/vpn-ui'), unit, menu):
                check('removed ' + str(path), not path.exists())
            check('operator notes retained', marker.read_text() == 'retain unrelated operator files\n')
            check('unowned host config retained', host_file.read_text() == '# operator-owned sentinel\n')
            check('firewall table removed', run(['nft', 'list', 'table', 'ip', 'vpn'], check=False).returncode != 0)
            check('policy rule removed', 'fwmark 0x1 lookup 100' not in run(['ip', 'rule', 'show']).stdout)
            check('policy routes removed', not run(['ip', 'route', 'show', 'table', '100'], check=False).stdout.strip())
            with socket.socket() as probe:
                probe.settimeout(.5)
                check('panel port released', probe.connect_ex(('127.0.0.1', PORT)) != 0)
        finally:
            if panel.poll() is None:
                panel.terminate()
                try:
                    panel.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    panel.kill()
                    panel.wait()
    print(json.dumps({'binary_sha256': digest, 'checks': CHECKS, 'coverage': 'container userland; no distro kernel or systemd boot coverage'}), flush=True)


if __name__ == '__main__':
    main()
