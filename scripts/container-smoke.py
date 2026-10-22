"""Exercise the production image against disposable volumes and credentials."""
import http.cookiejar
import json
import os
import pathlib
import re
import secrets
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import uuid

image = sys.argv[1]
prefix = "glance-smoke-" + uuid.uuid4().hex[:12]
containers = []
volumes = [prefix + suffix for suffix in ("-data", "-backups", "-restored")]


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True).strip()


def client():
    return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))


def api(opener, base, path, payload=None):
    headers = {"Origin": base}
    if payload is not None:
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(base + path, data=None if payload is None else json.dumps(payload).encode(), headers=headers)
    with opener.open(request, timeout=5) as response:
        return json.load(response)


with tempfile.TemporaryDirectory(prefix="glance-smoke-") as temporary:
    secret = pathlib.Path(temporary) / "owner_password"
    password = secrets.token_urlsafe(32)
    secret.write_text(password)
    secret.chmod(0o444)
    common = ["--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--memory=512m", "--pids-limit=128", "--tmpfs=/tmp:size=16m,mode=1777", "--mount", f"type=bind,src={secret},dst=/run/secrets/owner_password,readonly", "-e", "GLANCE_DASHBOARD_TOKEN_FILE=/run/secrets/owner_password"]
    if os.environ.get("CODEX_PROXY_CERT"):
        common += ["--mount", "type=bind,src=/etc/ssl/certs/ca-certificates.crt,dst=/run/cloud-ca.pem,readonly", "-e", "SSL_CERT_FILE=/run/cloud-ca.pem"]

    def start(name, volume):
        docker("run", "-d", "--name", name, *common, "-p", "127.0.0.1::8080", "-v", volume + ":/app/data", "-v", volumes[1] + ":/app/backups", image)
        containers.append(name)
        base = "http://" + docker("port", name, "8080/tcp")
        for attempt in range(60):
            try:
                state = api(client(), base, "/api/v1/auth/session")
                if state["enabled"]:
                    return base
            except (OSError, ValueError):
                pass
            time.sleep(0.25)
        raise RuntimeError("container did not become ready")

    try:
        for volume in volumes:
            docker("volume", "create", volume)
        name = prefix + "-app"
        base = start(name, volumes[0])
        assert docker("top", name, "-eo", "pid,uid").splitlines()[1].split()[1] == "10001"
        try:
            api(client(), base, "/api/v1/dashboard")
            raise AssertionError("anonymous dashboard access was allowed")
        except urllib.error.HTTPError as error:
            assert error.code == 401
        owner = client()
        api(owner, base, "/api/v1/auth/login", {"password": password})
        board = api(owner, base, "/api/v1/dashboard")
        board["title"] = "Container persistence rehearsal"
        request = urllib.request.Request(base + "/api/v1/dashboard", data=json.dumps(board).encode(), method="PUT", headers={"Content-Type": "application/json", "Origin": base})
        saved = json.load(owner.open(request, timeout=5))
        assert api(owner, base, "/api/readyz")["status"] == "ready"
        with owner.open(base + "/") as response:
            page = response.read().decode()
        assets = re.findall(r'(?:src|href)="(/assets/[^"]+)"', page)
        assert len(assets) >= 2
        for asset in assets:
            assert owner.open(base + asset).status == 200
        docker("exec", name, "/app/glance", "dashboard", "healthcheck")
        docker("exec", name, "/app/glance", "dashboard", "backup", "--db", "/app/data/dashboard.db", "--output", "/app/backups/rehearsal.db")
        docker("exec", name, "/app/glance", "dashboard", "verify-backup", "--input", "/app/backups/rehearsal.db")
        docker("restart", name)
        base = "http://" + docker("port", name, "8080/tcp")
        for attempt in range(60):
            try:
                owner = client()
                api(owner, base, "/api/v1/auth/login", {"password": password})
                break
            except OSError:
                time.sleep(0.25)
        else:
            raise RuntimeError("restarted container did not become ready")
        assert api(owner, base, "/api/v1/dashboard") == saved
        docker("run", "--rm", *common, "-v", volumes[2] + ":/app/data", "-v", volumes[1] + ":/app/backups:ro", image, "dashboard", "restore", "--input", "/app/backups/rehearsal.db", "--db", "/app/data/dashboard.db")
        restored = start(prefix + "-restore", volumes[2])
        restored_owner = client()
        api(restored_owner, restored, "/api/v1/auth/login", {"password": password})
        assert api(restored_owner, restored, "/api/v1/dashboard") == saved
        print("PASS: non-root/read-only image, authentication, assets, readiness, restart persistence and restore to a clean volume")
    finally:
        for name in reversed(containers):
            subprocess.run(["docker", "rm", "-f", name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for volume in volumes:
            subprocess.run(["docker", "volume", "rm", volume], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
