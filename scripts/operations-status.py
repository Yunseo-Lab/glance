#!/usr/bin/env python3
"""Read-only local operations check; output never includes credentials or DB rows."""
import argparse, datetime, json, pathlib, shutil, subprocess, time

p = argparse.ArgumentParser(description=__doc__)
p.add_argument("--binary", default="build/glance")
p.add_argument("--backups", default="backups")
p.add_argument("--data-directory", default=".")
p.add_argument("--max-backup-age-hours", type=int, default=48)
a = p.parse_args()
now = time.time()
backups = pathlib.Path(a.backups)
candidates = (
    sorted(backups.glob("dashboard-????????T??????Z-*.db"), reverse=True)
    if backups.is_dir()
    else []
)
valid = None
invalid = 0
for file in candidates[:1000]:
    if file.is_symlink() or not file.is_file():
        continue
    if (
        subprocess.run(
            [a.binary, "dashboard", "verify-backup", "--input", str(file)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        ).returncode
        == 0
    ):
        valid = file
        break
    invalid += 1
age = max(0, int(now - valid.stat().st_mtime)) if valid else None
disk = shutil.disk_usage(a.data_directory)
result = {
    "checked_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "backup": {
        "verified": bool(valid),
        "age_seconds": age,
        "newer_invalid_files": invalid,
        "warning": not valid or age > a.max_backup_age_hours * 3600 or invalid > 0,
    },
    "disk": {
        "free_bytes": disk.free,
        "total_bytes": disk.total,
        "warning": disk.free < max(1024**3, int(disk.total * 0.1)),
    },
}
print(json.dumps(result))
raise SystemExit(2 if result["backup"]["warning"] or result["disk"]["warning"] else 0)
