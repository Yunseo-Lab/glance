#!/usr/bin/env python3
"""Export a verified DB/key pair to separate private mounted destinations.
No network transfer is performed; mount your chosen offsite storage separately.
"""
import argparse, hashlib, json, os, pathlib, secrets, stat, subprocess

p = argparse.ArgumentParser(description=__doc__)
p.add_argument("--binary", default="build/glance")
p.add_argument("--backup", required=True)
p.add_argument("--key-file", required=True)
p.add_argument("--db-destination", required=True)
p.add_argument("--key-destination", required=True)
a = p.parse_args()
backup = pathlib.Path(a.backup)
key = pathlib.Path(a.key_file)
for source in [backup, key]:
    info = source.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077:
        raise SystemExit("Source must be a private regular file (0600/0400)")
dbdir = pathlib.Path(a.db_destination).resolve()
keydir = pathlib.Path(a.key_destination).resolve()
if dbdir == keydir or dbdir in keydir.parents or keydir in dbdir.parents:
    raise SystemExit(
        "Database and key destinations must be separate, non-nested directories"
    )
for directory in [dbdir, keydir]:
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    if directory.stat().st_mode & 0o077:
        raise SystemExit("Destination directory must be private (0700)")
subprocess.run(
    [
        a.binary,
        "dashboard",
        "verify-backup",
        "--input",
        str(backup),
        "--key-file",
        str(key),
    ],
    check=True,
    stdout=subprocess.DEVNULL,
)
identifier = secrets.token_hex(8)


def publish(source, target):
    # Exclusive creation preserves every earlier known-good export.
    digest = hashlib.sha256()
    with source.open("rb") as original, target.open("xb") as stream:
        os.chmod(target, 0o600)
        while chunk := original.read(1024 * 1024):
            digest.update(chunk)
            stream.write(chunk)
        stream.flush()
        os.fsync(stream.fileno())
    return digest.hexdigest()


result = {
    "id": identifier,
    "database_sha256": publish(backup, dbdir / (identifier + ".db")),
    "key_sha256": publish(key, keydir / (identifier + ".key")),
    "pair_verified": True,
    "transfer": "local/mounted destinations; confirm your provider replication separately",
}
for directory in [dbdir, keydir]:
    with (directory / (identifier + ".json")).open("x") as stream:
        os.chmod(stream.name, 0o600)
        json.dump(result, stream, indent=2)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())
    descriptor = os.open(directory, os.O_RDONLY)
    os.fsync(descriptor)
    os.close(descriptor)
print(json.dumps(result))
