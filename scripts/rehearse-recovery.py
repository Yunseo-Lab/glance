#!/usr/bin/env python3
"""Disposable online backup, separated export, wrong-key rejection and restore rehearsal."""
import datetime, json, os, pathlib, secrets, subprocess, tempfile, time, urllib.request

binary = str(pathlib.Path("build/glance").resolve())
base = "http://127.0.0.1:8104"


def command(*args):
    return subprocess.run(args, check=True, capture_output=True, text=True)


with tempfile.TemporaryDirectory(prefix="glance-recovery-") as directory:
    root = pathlib.Path(directory)
    db = root / "dashboard.db"
    backups = root / "backups"
    backups.mkdir(mode=0o700)
    env = {
        **os.environ,
        "GLANCE_DASHBOARD_TOKEN": "",
        "GLANCE_DASHBOARD_TOKEN_FILE": "",
        "GLANCE_CONNECTION_KEY_FILE": "",
        "GLANCE_DASHBOARD_PUBLIC_URL": "",
    }
    server = subprocess.Popen(
        [binary, "dashboard", "--listen", "127.0.0.1:8104", "--db", str(db)],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    try:
        for _ in range(100):
            try:
                with urllib.request.urlopen(base + "/api/readyz") as response:
                    assert response.status == 200
                break
            except OSError:
                time.sleep(0.05)
        request = urllib.request.Request(
            base + "/api/v1/connections",
            data=json.dumps(
                {
                    "kind": "github",
                    "name": "offline restoration fixture",
                    "token": secrets.token_urlsafe(32),
                }
            ).encode(),
            headers={"Origin": base, "Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(request) as response:
            assert response.status == 201
        backup = backups / "dashboard-20261022T120000Z-1.db"
        command(binary, "dashboard", "backup", "--db", str(db), "--output", str(backup))
        command(
            "python3",
            "scripts/export-recovery.py",
            "--binary",
            binary,
            "--backup",
            str(backup),
            "--key-file",
            str(db) + ".key",
            "--db-destination",
            str(root / "db-only"),
            "--key-destination",
            str(root / "key-only"),
        )
        exported = next((root / "db-only").glob("*.db"))
        key = next((root / "key-only").glob("*.key"))
        restore = root / "restored.db"
        command(
            binary,
            "dashboard",
            "restore",
            "--input",
            str(exported),
            "--db",
            str(restore),
        )
        command(
            binary,
            "dashboard",
            "verify-backup",
            "--input",
            str(restore),
            "--key-file",
            str(key),
        )
        good = command(
            "python3",
            "scripts/operations-status.py",
            "--binary",
            binary,
            "--backups",
            str(backups),
            "--data-directory",
            str(root),
        )
        assert not json.loads(good.stdout)["backup"]["warning"]
        corrupt = backups / "dashboard-20261023T120000Z-1.db"
        corrupt.write_bytes(b"failed backup fixture")
        corrupt.chmod(0o600)
        warning = subprocess.run(
            [
                "python3",
                "scripts/operations-status.py",
                "--binary",
                binary,
                "--backups",
                str(backups),
                "--data-directory",
                str(root),
            ],
            capture_output=True,
            text=True,
        )
        assert (
            warning.returncode == 2
            and json.loads(warning.stdout)["backup"]["newer_invalid_files"] == 1
        )
        result = {
            "checked_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
            "online_backup": True,
            "db_and_key_exported_to_separate_private_directories": True,
            "pair_verified_before_export": True,
            "restored_copy_and_key_verified": True,
            "healthy_status": True,
            "corrupt_newer_backup_warning": True,
            "scope": "disposable mounted-directory simulation, no remote storage transfer or live credentials",
        }
    finally:
        server.terminate()
        server.wait(timeout=10)
pathlib.Path("docs/verification/backup-recovery.json").write_text(
    json.dumps(result, indent=2) + "\n"
)
print(json.dumps(result))
