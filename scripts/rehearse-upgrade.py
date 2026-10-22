#!/usr/bin/env python3
"""Rehearse an actual v1 binary -> current binary -> old binary + old DB rollback.
All writes are confined to a new temporary directory; never uses deployment data.
"""
import argparse, hashlib, json, os, pathlib, sqlite3, subprocess, tempfile, time, urllib.request

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--old-binary", required=True)
parser.add_argument("--new-binary", default="build/glance")
parser.add_argument("--output", default="docs/verification/upgrade-rollback.json")
args = parser.parse_args()
old = str(pathlib.Path(args.old_binary).resolve())
new = str(pathlib.Path(args.new_binary).resolve())
env = {
    **os.environ,
    "GLANCE_DASHBOARD_TOKEN": "",
    "GLANCE_DASHBOARD_TOKEN_FILE": "",
    "GLANCE_CONNECTION_KEY_FILE": "",
    "GLANCE_DASHBOARD_PUBLIC_URL": "",
}
base = "http://127.0.0.1:8102"


def call(path, body=None):
    request = urllib.request.Request(
        base + path,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Content-Type": "application/json", "Origin": base},
        method="PUT" if body is not None else "GET",
    )
    with urllib.request.urlopen(request, timeout=5) as response:
        return json.load(response)


def start(binary, db):
    process = subprocess.Popen(
        [binary, "dashboard", "--listen", "127.0.0.1:8102", "--db", str(db)],
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.PIPE,
    )
    for _ in range(100):
        if process.poll() is not None:
            raise RuntimeError(process.stderr.read().decode())
        try:
            call("/api/readyz")
            return process
        except Exception:
            time.sleep(0.05)
    process.terminate()
    process.wait()
    raise RuntimeError("isolated server not ready")


def stop(process):
    process.terminate()
    process.wait(timeout=10)
    process.stderr.close()


def projection(board):
    return {
        key: board[key]
        for key in ["id", "title", "theme", "revision", "widgets", "sources"]
    }


with tempfile.TemporaryDirectory(prefix="glance-upgrade-") as directory:
    root = pathlib.Path(directory)
    db = root / "dashboard.db"
    process = start(old, db)
    try:
        board = call("/api/v1/dashboard")
        board["title"] = "v1 → 새 버전 → 복구"
        board["sources"] = []
        board["widgets"] = [
            {
                "id": "v1-clock",
                "type": "clock",
                "title": "기존 시계",
                "size": "small",
                "config": {"timezone": "UTC"},
            },
            {
                "id": "v1-note",
                "type": "note",
                "title": "기존 메모",
                "size": "wide",
                "config": {"body": "한글 원문과 **강조**를 보존합니다."},
            },
            {
                "id": "v1-links",
                "type": "bookmarks",
                "title": "기존 링크",
                "size": "small",
                "config": {
                    "links": [
                        {
                            "title": "Glance",
                            "url": "https://github.com/Yunseo-Lab/glance",
                        }
                    ]
                },
            },
        ]
        call("/api/v1/dashboard", board)
        before = call("/api/v1/dashboard")
    finally:
        stop(process)
    with sqlite3.connect(db) as connection:
        assert connection.execute("PRAGMA user_version").fetchone()[0] == 1
    process = start(new, db)
    try:
        after = call("/api/v1/dashboard")
        # Version-1 omission remains supported; compare the actual preserved documents.
        assert projection(after) == projection(before), (after, before)
        assert call("/api/v1/capabilities")["fts5"] is True
    finally:
        stop(process)
    with sqlite3.connect(db) as connection:
        version = connection.execute("PRAGMA user_version").fetchone()[0]
    backups = list(root.glob("dashboard.db.before-v*.db"))
    assert len(backups) == 1
    subprocess.run(
        [new, "dashboard", "verify-backup", "--input", str(backups[0])],
        check=True,
        capture_output=True,
        env=env,
    )
    original_hash = hashlib.sha256(db.read_bytes()).hexdigest()
    refused = subprocess.run(
        [old, "dashboard", "--listen", "127.0.0.1:8102", "--db", str(db)],
        env=env,
        capture_output=True,
        timeout=10,
    )
    assert refused.returncode != 0 and b"unsupported database version" in refused.stderr
    assert hashlib.sha256(db.read_bytes()).hexdigest() == original_hash
    restored = root / "rollback.db"
    subprocess.run(
        [
            new,
            "dashboard",
            "restore",
            "--input",
            str(backups[0]),
            "--db",
            str(restored),
        ],
        check=True,
        capture_output=True,
        env=env,
    )
    process = start(old, restored)
    try:
        assert projection(call("/api/v1/dashboard")) == projection(before)
    finally:
        stop(process)
    result = {
        "historical_source_commit": "0a376f9",
        "old_binary_sha256": hashlib.sha256(pathlib.Path(old).read_bytes()).hexdigest(),
        "new_binary_sha256": hashlib.sha256(pathlib.Path(new).read_bytes()).hexdigest(),
        "from_schema": 1,
        "to_schema": version,
        "fixture": "real v1 server writes three local widgets, Unicode text and bookmarks; no credentials or upstream calls",
        "document_ids_and_config_preserved": True,
        "automatic_pre_upgrade_backup_verified": True,
        "old_binary_refuses_new_schema_without_data_change": True,
        "old_binary_with_restored_old_database": True,
    }
    pathlib.Path(args.output).write_text(
        json.dumps(result, indent=2, ensure_ascii=False) + "\n"
    )
    print(json.dumps(result, ensure_ascii=False))
