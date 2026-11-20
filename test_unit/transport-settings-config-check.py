#!/usr/bin/env python3
"""Validate XHTTP, TLS/REALITY, sockopt, DNS, routing and sniffing with pinned Xray.

Usage: python3 test_unit/transport-settings-config-check.py /path/to/xray
The script uses temporary loopback configs/certificates and makes no listeners.
"""
import base64
import json
from pathlib import Path
import secrets
import subprocess
import sys
import tempfile


XRAY = str(Path(sys.argv[1]).resolve())


def check_config(binary, root, name, config):
    path = root / f'{name}.json'
    path.write_text(json.dumps(config))
    p = subprocess.run([binary, 'run', '-test', '-c', str(path)], capture_output=True,
                       text=True, timeout=45)
    if p.returncode:
        raise SystemExit(f'FAIL {name}:\n{p.stdout}\n{p.stderr}')
    print(f'PASS pinned Xray {name} configuration', flush=True)


def main():
    with tempfile.TemporaryDirectory(prefix='vpn-aio-transport-audit-') as temp:
        root = Path(temp)
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1',
                        '-subj', '/CN=localhost', '-keyout', str(root / 'key.pem'),
                        '-out', str(root / 'cert.pem')], check=True, capture_output=True, timeout=30)
        key = base64.urlsafe_b64encode(secrets.token_bytes(32)).decode().rstrip('=')
        sockopt = {
            'tcpFastOpen': True, 'tcpKeepAliveInterval': 30, 'tcpKeepAliveIdle': 300,
            'tcpMaxSeg': 1400, 'tcpUserTimeout': 10000, 'tcpWindowClamp': 600,
            'tcpCongestion': 'cubic', 'tcpMptcp': False, 'domainStrategy': 'UseIPv4',
            'addressPortStrategy': 'srvportandaddress',
            'happyEyeballs': {'prioritizeIPv6': True, 'tryDelayMs': 20, 'interleave': 1, 'maxConcurrentTry': 4},
            'customSockopt': [{'system': 'linux', 'network': 'tcp', 'level': 'socket', 'opt': 'reuseaddr',
                               'value': '1', 'type': 'int'}],
        }
        dns = {
            'servers': [{'address': '1.1.1.1', 'port': 53, 'domains': ['domain:example.com'],
                         'expectedIPs': ['geoip:private'], 'queryStrategy': 'UseIPv4',
                         'disableCache': False, 'serveStale': True, 'serveExpiredTTL': 60,
                         'finalQuery': True}],
            'queryStrategy': 'UseIPv4', 'disableFallback': True,
            'disableFallbackIfMatch': True, 'enableParallelQuery': True, 'useSystemHosts': True,
            'tag': 'dns-in', 'clientIp': '127.0.0.1',
        }
        routing = {
            'domainStrategy': 'IPIfNonMatch',
            'rules': [
                {'type': 'field', 'domain': ['domain:example.com'], 'outboundTag': 'direct'},
                {'type': 'field', 'source': ['10.0.0.0/8'], 'outboundTag': 'direct'},
            ],
        }
        tls = {
            'serverName': 'localhost', 'alpn': ['h2', 'http/1.1'],
            'minVersion': '1.2', 'maxVersion': '1.3', 'cipherSuites': '',
            'rejectUnknownSni': False, 'disableSystemRoot': False,
            'enableSessionResumption': True, 'curvePreferences': ['X25519', 'CurveP256'],
            'useSystemCA': True, 'masterKeyLog': 'none',
            'certificates': [{'certificateFile': str(root / 'cert.pem'), 'keyFile': str(root / 'key.pem'),
                              'usage': 'encipherment', 'ocspStapling': 3600}],
        }
        xhttp = {
            'host': 'localhost', 'path': '/xhttp', 'mode': 'packet-up', 'headers': {'X-Test': 'ok'},
            'uplinkHTTPMethod': 'POST', 'uplinkDataPlacement': 'cookie', 'uplinkDataKey': 'x_data',
            'sessionIDPlacement': 'query', 'sessionIDKey': 'x_session',
            'seqPlacement': 'header', 'seqKey': 'X-Seq', 'xPaddingBytes': '100-200',
            'xPaddingObfsMode': True, 'scMaxBufferedPosts': 30,
            'scMaxEachPostBytes': '1000000', 'scStreamUpServerSecs': '20-80',
            'scMinPostsIntervalMs': '30', 'serverMaxHeaderBytes': 0,
            'xmux': {'maxConcurrency': '16-32', 'maxConnections': 0, 'cMaxReuseTimes': 0,
                     'hMaxRequestTimes': '600-900', 'hMaxReusableSecs': '1800-3000',
                     'hKeepAlivePeriod': 0},
        }
        inbound = {'listen': '127.0.0.1', 'port': 19443, 'protocol': 'vless',
                   'settings': {'clients': [{'id': '00000000-0000-4000-8000-000000000001'}], 'decryption': 'none'},
                   'streamSettings': {'network': 'xhttp', 'security': 'tls', 'tlsSettings': tls,
                                      'xhttpSettings': xhttp, 'sockopt': sockopt},
                   'sniffing': {'enabled': True, 'destOverride': ['http', 'tls', 'quic'],
                                'metadataOnly': True, 'routeOnly': True,
                                'ipsExcluded': ['geoip:private'], 'domainsExcluded': ['domain:example.org']}}
        config = {'log': {'loglevel': 'warning'}, 'dns': dns, 'routing': routing,
                  'inbounds': [inbound], 'outbounds': [
                      {'tag': 'direct', 'protocol': 'freedom', 'settings': {'domainStrategy': 'UseIPv4'}},
                      {'tag': 'blocked', 'protocol': 'blackhole'},
                  ]}
        check_config(XRAY, root, 'xhttp-tls-sockopt-dns-routing-sniffing', config)

        reality = {
            'show': True, 'xver': 0, 'target': '127.0.0.1:443', 'type': 'tcp',
            'serverNames': ['localhost'], 'privateKey': key, 'minClientVer': '',
            'maxClientVer': '', 'maxTimeDiff': 0, 'shortIds': ['0123456789abcdef'],
            'settings': {'fingerprint': 'chrome', 'mldsa65Verify': ''},
        }
        reality_config = {'log': {'loglevel': 'warning'}, 'dns': dns, 'routing': routing,
                          'inbounds': [{'listen': '127.0.0.1', 'port': 19444, 'protocol': 'vless',
                                        'settings': {'clients': [{'id': '00000000-0000-4000-8000-000000000001'}],
                                                     'decryption': 'none'},
                                        'streamSettings': {'network': 'xhttp', 'security': 'reality',
                                                           'realitySettings': reality, 'xhttpSettings': xhttp,
                                                           'sockopt': sockopt},
                                        'sniffing': inbound['sniffing']}],
                          'outbounds': config['outbounds']}
        check_config(XRAY, root, 'xhttp-reality-sockopt-dns-routing-sniffing', reality_config)

        client_user = {'id': '00000000-0000-4000-8000-000000000001', 'encryption': 'none'}
        client_settings = {'vnext': [{'address': 'proxy.example.test', 'port': 443,
                                      'users': [client_user]}]}
        outbound_xhttp = {'path': '/xhttp', 'host': 'proxy.example.test', 'mode': 'stream-one',
                          'headers': {'X-Test': 'outbound'}, 'xPaddingBytes': '100-200',
                          'xmux': {'maxConcurrency': '16-32', 'maxConnections': 0}}
        tls_outbound = {'log': {'loglevel': 'warning'}, 'inbounds': [],
                        'outbounds': [
                            {'tag': 'proxy', 'protocol': 'vless', 'settings': client_settings,
                             'streamSettings': {'network': 'xhttp', 'security': 'tls',
                                                'tlsSettings': {'serverName': 'proxy.example.test',
                                                                'alpn': ['h2'], 'fingerprint': 'chrome',
                                                                'verifyPeerCertByName': 'proxy.example.test',
                                                                'echConfigList': ''},
                                                'xhttpSettings': outbound_xhttp,
                                                'sockopt': sockopt}},
                            {'protocol': 'freedom'},
                        ]}
        check_config(XRAY, root, 'outbound-xhttp-tls-sockopt', tls_outbound)

        reality_public = base64.urlsafe_b64encode(secrets.token_bytes(32)).decode().rstrip('=')
        reality_outbound = {'log': {'loglevel': 'warning'}, 'inbounds': [],
                            'outbounds': [
                                {'tag': 'proxy', 'protocol': 'vless', 'settings': client_settings,
                                 'streamSettings': {'network': 'xhttp', 'security': 'reality',
                                                    'realitySettings': {'serverName': 'proxy.example.test',
                                                                        'publicKey': reality_public,
                                                                        'shortId': '0123456789abcdef',
                                                                        'fingerprint': 'chrome',
                                                                        'spiderX': '/'},
                                                    'xhttpSettings': outbound_xhttp,
                                                    'sockopt': sockopt}},
                                {'protocol': 'freedom'},
                            ]}
        check_config(XRAY, root, 'outbound-xhttp-reality-sockopt', reality_outbound)


if __name__ == '__main__':
    main()
