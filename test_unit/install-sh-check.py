#!/usr/bin/env python3
"""Offline contract check for install.sh (version-pinned one-liner installer).

Runs the script the way curl does (`bash script ARGS`), stubbing curl so no
network is touched, and asserts: --help text, version validation, exit codes,
and that a pinned version is forwarded to deploy.sh as VPN_AIO_VERSION.
"""
import pathlib
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
INSTALL_SH = ROOT / "install.sh"

FAKE_DEPLOY = """#!/usr/bin/env bash
set -euo pipefail
printf 'stub deploy: version=%s\\\\n' "${VPN_AIO_VERSION:-latest}"
"""


def run(args, env=None):
    with tempfile.NamedTemporaryFile("w", suffix=".sh", delete=False) as f:
        f.write(FAKE_DEPLOY)
        stub = f.name
    env = dict(env or {})
    # The installer's own curl is stubbed: -fsSL URL -o FILE must write the stub.
    env["PATH"] = str(pathlib.Path(stub).parent) + ":/usr/bin:/bin"
    curl_stub = pathlib.Path(pathlib.Path(stub).parent) / "curl"
    curl_stub.write_text(
        "#!/usr/bin/env bash\n"
        'out=""\nwhile [[ $# -gt 0 ]]; do case "$1" in -o) out="$2"; shift 2;; *) shift;; esac; done\n'
        f"cp {stub} \"$out\"\n"
    )
    curl_stub.chmod(0o755)
    try:
        return subprocess.run(
            ["bash", str(INSTALL_SH), *args],
            capture_output=True, text=True, env=env, timeout=30,
        )
    finally:
        pathlib.Path(stub).unlink()
        curl_stub.unlink()


def expect(name, proc, code, needle=None):
    ok = proc.returncode == code and (needle is None or needle in (proc.stdout + proc.stderr))
    print(f"{'PASS' if ok else 'FAIL'}  {name} (exit={proc.returncode})")
    if not ok:
        print(proc.stdout)
        print(proc.stderr)
        sys.exit(1)


expect("--help", run(["--help"]), 0, "VPN-AIO installer")
expect("-h", run(["-h"]), 0, "Usage:")
expect("too many args", run(["v1.0.0", "v2.0.0"]), 2, "usage:")
expect("invalid version", run(["not-a-version"]), 2, "invalid version")
expect("invalid version 3.7", run(["3.7"]), 2, "invalid version")

p = run(["v3.1.0"])
expect("pinned version forwarded", p, 0, "stub deploy: version=v3.1.0")
p = run(["3.0.0"])
expect("version without v normalized", p, 0, "stub deploy: version=v3.0.0")
p = run([])
expect("no args installs latest", p, 0, "stub deploy: version=latest")

print("install-sh-check: all checks passed")
