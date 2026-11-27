#!/usr/bin/env python3
"""Regression check: deploy.sh must never prompt without a usable terminal.

`[[ -r /dev/tty ]]` is true even with no controlling terminal (cloud-init, CI,
setsid), which made unattended installs hang or die on ENXIO. This check:
  1. asserts deploy.sh no longer gates prompts on `-r /dev/tty`;
  2. extracts deploy_have_tty and runs it under `setsid` with stdin /dev/null,
     expecting "no terminal";
  3. asserts the PANEL_USER/PANEL_PASS/PANEL_PORT/PANEL_PATH unattended
     credential branch is present and selected before the tty prompt.
Linux only (needs setsid). No network, no root.
"""
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
DEPLOY = (ROOT / "deploy.sh").read_text().replace("\r\n", "\n")


def check(name, ok, detail=""):
    print(f"{'PASS' if ok else 'FAIL'}  {name}")
    if not ok:
        if detail:
            print(detail)
        sys.exit(1)


check("no prompt gated on -r /dev/tty", "[[ -r /dev/tty ]]" not in DEPLOY)

m = re.search(r"^deploy_have_tty\(\) \{.*\}$", DEPLOY, re.M)
check("deploy_have_tty defined", m is not None)

script = m.group(0) + "\ndeploy_have_tty && echo TTY || echo NO_TTY\n"
p = subprocess.run(["setsid", "bash", "-c", script], stdin=subprocess.DEVNULL,
                   capture_output=True, text=True, timeout=30)
check("no controlling terminal detected under setsid",
      p.stdout.strip() == "NO_TTY", p.stdout + p.stderr)

env_idx = DEPLOY.find('cred_mode="env"')
tty_idx = DEPLOY.find("elif deploy_have_tty; then", DEPLOY.find('cred_mode="random"'))
check("PANEL_* env credential mode precedes tty prompt",
      env_idx != -1 and tty_idx != -1 and env_idx < tty_idx)
check("env mode applies PANEL_* values",
      '--user "${PANEL_USER:-}" --pass "${PANEL_PASS:-}"' in DEPLOY)

print("deploy-tty-check: all checks passed")
