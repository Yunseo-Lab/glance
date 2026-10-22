#!/usr/bin/env python3
"""Verify local HTTPS through the patched Caddy; no public DNS/ACME claim."""
import http.cookiejar, json, os, pathlib, secrets, ssl, subprocess, tempfile, time, urllib.request

base = "https://localhost:8445"
binary = str(pathlib.Path("build/glance").resolve())
proxy = str(pathlib.Path("build/caddy").resolve())
with tempfile.TemporaryDirectory(prefix="glance-https-") as directory:
    root = pathlib.Path(directory)
    token = secrets.token_urlsafe(32)
    config = root / "Caddyfile"
    config.write_text(
        "{\n admin off\n auto_https disable_redirects\n skip_install_trust\n}\nhttps://localhost:8445 {\n bind 127.0.0.1\n tls internal\n reverse_proxy 127.0.0.1:8105\n}\n"
    )
    env = {
        **os.environ,
        "GLANCE_DASHBOARD_TOKEN": token,
        "GLANCE_DASHBOARD_TOKEN_FILE": "",
        "GLANCE_CONNECTION_KEY_FILE": "",
        "GLANCE_DASHBOARD_PUBLIC_URL": base,
        "XDG_DATA_HOME": str(root / "data"),
        "XDG_CONFIG_HOME": str(root / "config"),
        "XDG_CACHE_HOME": str(root / "cache"),
    }
    logs = [(root / name).open("w+") for name in ["app.log", "proxy.log"]]
    processes = [
        subprocess.Popen(
            [
                binary,
                "dashboard",
                "--listen",
                "127.0.0.1:8105",
                "--db",
                str(root / "dashboard.db"),
            ],
            env=env,
            stdout=logs[0],
            stderr=logs[0],
        ),
        subprocess.Popen(
            [proxy, "run", "--config", str(config), "--adapter", "caddyfile"],
            env=env,
            stdout=logs[1],
            stderr=logs[1],
        ),
    ]
    try:
        ca = root / "data/caddy/pki/authorities/local/root.crt"
        for _ in range(200):
            if any(process.poll() is not None for process in processes):
                raise RuntimeError(
                    "isolated TLS process exited; inspect local rehearsal log"
                )
            if ca.exists():
                break
            time.sleep(0.05)
        context = ssl.create_default_context(cafile=str(ca))
        cookies = http.cookiejar.CookieJar()
        client = urllib.request.build_opener(
            urllib.request.HTTPSHandler(context=context),
            urllib.request.HTTPCookieProcessor(cookies),
        )

        def request(path, body=None, method=None):
            req = urllib.request.Request(
                base + path,
                data=json.dumps(body).encode() if body is not None else None,
                headers={"Origin": base, "Content-Type": "application/json"},
                method=method,
            )
            with client.open(req, timeout=10) as response:
                return json.load(response)

        for _ in range(100):
            try:
                request("/api/v1/auth/session")
                break
            except OSError:
                time.sleep(0.05)
        try:
            request("/api/v1/dashboard")
            raise AssertionError("anonymous API accepted")
        except urllib.error.HTTPError as error:
            assert error.code == 401
        request("/api/v1/auth/login", {"password": token})
        cookie = next(iter(cookies))
        assert (
            cookie.secure
            and cookie.has_nonstandard_attr("HttpOnly")
            and cookie.get_nonstandard_attr("SameSite") == "Strict"
        )
        board = request("/api/v1/dashboard")
        board["title"] = "로컬 HTTPS 저장 리허설"
        request("/api/v1/dashboard", board, "PUT")
        assert request("/api/v1/dashboard")["title"] == board["title"]
        request("/api/v1/auth/logout", {}, "POST")
        assert not request("/api/v1/auth/session")["authenticated"]
        result = {
            "transport": "local loopback HTTPS, Caddy internal CA verified by client",
            "public_dns_acme_or_phone": False,
            "anonymous_api_rejected": True,
            "secure_httponly_samesite_cookie": True,
            "authenticated_save": True,
            "logout_revokes_session": True,
            "proxy_binary_sha256": __import__("hashlib")
            .sha256(pathlib.Path(proxy).read_bytes())
            .hexdigest(),
        }
    finally:
        for process in processes:
            process.terminate()
        for process in processes:
            process.wait(timeout=10)
        for log in logs:
            log.seek(0)
            assert token not in log.read()
            log.close()
pathlib.Path("docs/verification/https-rehearsal.json").write_text(
    json.dumps(result, indent=2) + "\n"
)
print(json.dumps(result))
