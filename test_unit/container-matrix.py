#!/usr/bin/env python3
"""Run the container userland lifecycle matrix with an existing full binary.

Usage (Linux with Docker access):
  python3 test_unit/container-matrix.py --binary build/out/VPN-AIO-amd64
  python3 test_unit/container-matrix.py --binary PATH --only ubuntu-24,debian-12

No host mounts, privileged containers, module loading or host namespaces are used.
Package installation uses the container network; validation runs after disconnect.
This cannot replace the Incus VPN/kernel/systemd test matrix.
"""
import argparse
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import subprocess
import uuid


DISTROS = {
    'ubuntu-24': ('ubuntu:24.04', 'apt'),
    'ubuntu-26': ('ubuntu:26.04', 'apt'),
    'debian-12': ('debian:12', 'apt'),
    'debian-13': ('debian:13', 'apt'),
    'fedora-43': ('fedora:43', 'dnf'),
    'fedora-44': ('fedora:44', 'dnf'),
    'alma-9': ('almalinux:9', 'dnf'),
    'alma-10': ('almalinux:10', 'dnf'),
    'rocky-9': ('rockylinux:9', 'dnf'),
    'rocky-10': ('rockylinux:10', 'dnf'),
    'centos-9': ('quay.io/centos/centos:stream9', 'dnf'),
    'centos-10': ('quay.io/centos/centos:stream10', 'dnf'),
    'arch': ('archlinux:latest', 'pacman'),
}
INSTALL = {
    'apt': 'apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends python3 openssl ca-certificates nftables iproute2 procps',
    'dnf': 'dnf -y --setopt=install_weak_deps=False install python3 openssl ca-certificates nftables iproute procps-ng',
    'pacman': 'pacman -Syu --noconfirm python openssl ca-certificates nftables iproute2 procps-ng',
}
SCRIPTS = ('container-lifecycle.py', 'native-core-smoke.py', 'xdrive-config-check.py', 'finalmask-config-check.py', 'transport-settings-config-check.py')


def docker(*args, timeout=300):
    return subprocess.run(['docker', *map(str, args)], capture_output=True, text=True, timeout=timeout)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True, type=Path)
    parser.add_argument('--only', default=','.join(DISTROS))
    parser.add_argument('--output', type=Path)
    parser.add_argument('--expected-version', default=None,
                        help='candidate version; defaults to config/version in the checkout')
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    selected = args.only.split(',')
    if any(name not in DISTROS for name in selected):
        parser.error('--only must use names from: ' + ','.join(DISTROS))
    root = Path(__file__).resolve().parent
    for script in SCRIPTS:
        (root / script).resolve(strict=True)
    expected_version = args.expected_version
    if expected_version is None:
        version_file = root.parent / 'config' / 'version'
        expected_version = version_file.read_text(encoding='utf-8').strip()
    if not expected_version:
        parser.error('expected candidate version is empty')
    output = args.output or root / 'results' / ('containers-' + datetime.now(timezone.utc).strftime('%Y%m%d-%H%M%S'))
    output.mkdir(parents=True, exist_ok=False)
    with binary.open('rb') as source:
        digest = hashlib.file_digest(source, 'sha256').hexdigest()
    report = {'binary_sha256': digest,
              'coverage': 'container userland only; no VM/systemd/kernel qualification', 'results': []}
    try:
        for distro in selected:
            image, pm = DISTROS[distro]
            name = 'vpn-aio-lifecycle-' + uuid.uuid4().hex[:12]
            row = {'distro': distro, 'image': image, 'status': 'error'}
            report['results'].append(row)
            logs = []
            print('RUN ' + distro, flush=True)
            try:
                stages = [
                    ('pull image', ('pull', image), 300),
                    ('create container', ('run', '-d', '--name', name, '--memory=768m', '--cpus=1', '--pids-limit=256', '--cap-add=NET_ADMIN', image, 'sleep', 'infinity'), 60),
                    ('install test prerequisites', ('exec', name, 'bash', '-lc', INSTALL[pm]), 600),
                    ('disconnect network', ('network', 'disconnect', 'bridge', name), 30),
                    ('create test layout', ('exec', name, 'mkdir', '-p', '/checks', '/opt/VPN-AIO'), 30),
                    ('copy binary', ('cp', binary, name + ':/opt/VPN-AIO/VPN-AIO-amd64'), 90),
                ]
                stages += [('copy ' + script, ('cp', root / script, name + ':/checks/' + script), 30) for script in SCRIPTS]
                stages += [('lifecycle', ('exec', '-e', 'VPN_AIO_CONTAINER_LIFECYCLE_ACK=yes',
                                          '-e', 'VPN_AIO_EXPECTED_VERSION=' + expected_version,
                                          name, 'python3', '/checks/container-lifecycle.py'), 600)]
                for stage, command, timeout in stages:
                    proc = docker(*command, timeout=timeout)
                    logs.append(f'--- {stage} (exit {proc.returncode}) ---\n{proc.stdout}\n{proc.stderr}')
                    if proc.returncode:
                        row.update(status='fail' if stage == 'lifecycle' else 'error', stage=stage)
                        break
                else:
                    row['status'] = 'pass'
                    for line in proc.stdout.splitlines():
                        if line.startswith('{"binary_sha256"'):
                            row['lifecycle'] = json.loads(line)
                row['image_id'] = docker('image', 'inspect', image, '--format', '{{.Id}}', timeout=30).stdout.strip()
            except (OSError, subprocess.SubprocessError, ValueError) as exc:
                row.update(status='error', detail=str(exc))
            finally:
                # Collect only test-container logs, then remove only its unique name.
                for log in ('panel.log', 'uninstall.log'):
                    docker('cp', name + ':/tmp/' + log, output / (distro + '-' + log), timeout=30)
                cleanup = docker('rm', '-f', name, timeout=60)
                if cleanup.returncode and 'No such container' not in cleanup.stderr:
                    row['cleanup_error'] = cleanup.stderr
                (output / (distro + '.log')).write_text('\n'.join(logs))
                (output / 'results.json').write_text(json.dumps(report, indent=2))
            print(row['status'].upper() + ' ' + distro + (' (' + row['stage'] + ')' if 'stage' in row else ''), flush=True)
    finally:
        (output / 'results.json').write_text(json.dumps(report, indent=2))
    print('Results: ' + str(output), flush=True)
    return int(any(row['status'] != 'pass' or row.get('cleanup_error') for row in report['results']))


if __name__ == '__main__':
    raise SystemExit(main())
