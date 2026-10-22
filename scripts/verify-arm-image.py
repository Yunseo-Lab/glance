#!/usr/bin/env python3
"""Execute the actual ARM64 image's binary and libraries through local QEMU.
This is emulation evidence, not a Mac mini or native ARM hardware acceptance.
"""
import argparse, hashlib, json, os, pathlib, secrets, subprocess, tarfile, tempfile, time, urllib.request

p = argparse.ArgumentParser(description=__doc__)
p.add_argument("--image", default="glance-dashboard:frontier-arm64")
p.add_argument("--qemu", required=True)
p.add_argument("--output", default="docs/verification/arm64-image.json")
a = p.parse_args()
name = "glance-arm-evidence-" + secrets.token_hex(5)


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True).strip()


with tempfile.TemporaryDirectory(prefix="glance-arm64-") as directory:
    root = pathlib.Path(directory)
    fs = root / "rootfs"
    fs.mkdir()
    try:
        docker("create", "--name", name, "--platform", "linux/arm64", a.image)
        docker("export", "-o", str(root / "rootfs.tar"), name)
        with tarfile.open(root / "rootfs.tar") as archive:
            members = [
                member
                for member in archive
                if not (member.issym() and member.linkname.startswith("/"))
            ]
            archive.extractall(fs, members=members, filter="data")
    finally:
        subprocess.run(
            ["docker", "rm", "-f", name], check=False, stdout=subprocess.DEVNULL
        )
    token = secrets.token_urlsafe(32)
    env = {
        **os.environ,
        "GLANCE_DASHBOARD_TOKEN": token,
        "GLANCE_DASHBOARD_TOKEN_FILE": "",
        "GLANCE_CONNECTION_KEY_FILE": "",
        "GLANCE_DASHBOARD_PUBLIC_URL": "",
    }
    process = subprocess.Popen(
        [
            a.qemu,
            "-L",
            str(fs),
            str(fs / "app/glance"),
            "dashboard",
            "--listen",
            "127.0.0.1:8103",
            "--db",
            str(root / "dashboard.db"),
        ],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
    )

    def get(path, auth=True):
        import base64

        headers = (
            {
                "Authorization": "Basic "
                + base64.b64encode(("owner:" + token).encode()).decode()
            }
            if auth
            else {}
        )
        with urllib.request.urlopen(
            urllib.request.Request("http://127.0.0.1:8103" + path, headers=headers),
            timeout=10,
        ) as response:
            return json.load(response)

    try:
        for _ in range(200):
            if process.poll() is not None:
                raise RuntimeError(process.stderr.read().decode())
            try:
                get("/api/readyz")
                break
            except Exception:
                time.sleep(0.1)
        else:
            raise RuntimeError("emulated ARM server did not become ready")
        caps = get("/api/v1/capabilities")
        assert caps["fts5"] is True
        board = get("/api/v1/dashboard")
        assert board["id"] == "home"
        try:
            get("/api/v1/dashboard", False)
            raise AssertionError("unauthenticated API accepted")
        except urllib.error.HTTPError as error:
            assert error.code == 401
        result = {
            "image_id": docker("image", "inspect", "--format", "{{.Id}}", a.image),
            "architecture": docker(
                "image", "inspect", "--format", "{{.Architecture}}", a.image
            ),
            "execution": "QEMU user-mode with actual image rootfs libraries; not hardware",
            "binary_sha256": hashlib.sha256(
                (fs / "app/glance").read_bytes()
            ).hexdigest(),
            "fts5": True,
            "readiness": True,
            "owner_api": True,
            "anonymous_api_rejected": True,
            "sqlite": caps,
        }
    finally:
        process.terminate()
        process.wait(timeout=10)
        process.stderr.close()
    pathlib.Path(a.output).write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result))
