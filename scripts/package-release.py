#!/usr/bin/env python3
"""Package a reviewed native build with source, checksums and license provenance."""
import argparse, datetime, hashlib, json, os, pathlib, subprocess, tarfile

p = argparse.ArgumentParser(description=__doc__)
p.add_argument("--binary", default="build/glance")
p.add_argument("--output", default="release-artifacts")
p.add_argument("--name", choices=["glance", "caddy"], default="glance")
p.add_argument("--module-directory", default=".")
p.add_argument("--no-frontend", action="store_true")
a = p.parse_args()


def run(*args):
    return subprocess.check_output(args, text=True).strip()


commit = run("git", "rev-parse", "HEAD")
if run("git", "status", "--porcelain"):
    raise SystemExit(
        "Commit reviewed changes before packaging; refusing a source/binary mismatch"
    )
root = pathlib.Path(a.output)
root.mkdir(parents=True, exist_ok=True)
binary = pathlib.Path(a.binary)
metadata = run("go", "version", "-m", str(binary))
# Match tool output fields without relying on whitespace indentation.
os_name = next(
    line.split("GOOS=")[1].strip() for line in metadata.splitlines() if "GOOS=" in line
)
arch = next(
    line.split("GOARCH=")[1].strip()
    for line in metadata.splitlines()
    if "GOARCH=" in line
)
if f"vcs.revision={commit}" not in metadata or "vcs.modified=true" in metadata:
    raise SystemExit("Binary must be rebuilt from this clean commit")
archive = root / f"{a.name}-{os_name}-{arch}.tar.gz"
source = root / f"glance-source-{commit[:12]}.tar.gz"
subprocess.run(
    [
        "git",
        "archive",
        "--format=tar.gz",
        "--prefix=glance-source/",
        "--output",
        str(source),
        commit,
    ],
    check=True,
)
modules = []
for line in metadata.splitlines():
    fields = line.split()
    if fields and fields[0] == "dep":
        modules.append(
            {
                "module": fields[1],
                "version": fields[2],
                "go_sum": fields[3] if len(fields) > 3 else "",
            }
        )
lock = json.loads(pathlib.Path("web/package-lock.json").read_text())
frontend = [
    {
        "package": name.removeprefix("node_modules/"),
        "version": item.get("version"),
        "license": item.get("license", "see package source"),
        "integrity": item.get("integrity"),
    }
    for name, item in lock["packages"].items()
    if name and not item.get("dev")
]
if a.no_frontend:
    frontend = []
license_root = root / "licenses"
license_root.mkdir(exist_ok=True)


def notices(directory, target):
    target.mkdir(parents=True, exist_ok=True)
    found = []
    for file in pathlib.Path(directory).iterdir():
        if file.is_file() and file.name.lower().startswith(
            ("license", "copying", "notice", "copyright")
        ):
            content = file.read_bytes()
            (target / file.name).write_bytes(content)
            found.append(
                {
                    "file": str((target / file.name).relative_to(root)),
                    "sha256": hashlib.sha256(content).hexdigest(),
                }
            )
    return found


raw = subprocess.check_output(
    ["go", "list", "-m", "-json", "all"], cwd=a.module_directory, text=True
).strip()
decoder = json.JSONDecoder()
module_dirs = {}
while raw.strip():
    item, offset = decoder.raw_decode(raw.lstrip())
    raw = raw.lstrip()[offset:]
    module_dirs[item["Path"]] = item.get("Dir")
for module in modules:
    directory = module_dirs.get(module["module"])
    module["notices"] = (
        notices(directory, license_root / "go" / module["module"].replace("/", "__"))
        if directory
        else []
    )
    if not module["notices"]:
        raise SystemExit("Review missing Go license notice: " + module["module"])
for package in frontend:
    package["notices"] = notices(
        pathlib.Path("web/node_modules") / package["package"],
        license_root / "npm" / package["package"].replace("/", "__"),
    )
    if not package["notices"]:
        raise SystemExit("Review missing npm license notice: " + package["package"])
notices(run("go", "env", "GOROOT"), license_root / "go-standard-library")
with tarfile.open(archive, "w:gz") as tar:
    for file in [
        binary,
        (
            pathlib.Path("deploy/proxy/LICENSE")
            if a.name == "caddy"
            else pathlib.Path("LICENSE")
        ),
        pathlib.Path("docs/operations.md"),
        pathlib.Path("docs/pwa-offline.md"),
        license_root,
    ]:
        tar.add(file, arcname=a.name if file == binary else file.name)
licenses = {
    "project": (
        "Apache-2.0"
        if a.name == "caddy"
        else "AGPL-3.0-only; full LICENSE and corresponding source included"
    ),
    "go_modules": modules,
    "frontend_packages": frontend,
    "note": "Dependency inventory records upstream package license declarations; review upstream notices when redistributing. Runtime OS SBOM is generated separately by Trivy.",
}
(root / "dependency-provenance.json").write_text(json.dumps(licenses, indent=2) + "\n")
(root / "build-info.txt").write_text(metadata + "\n")
manifest = {
    "commit": commit,
    "source_url": f"https://github.com/Yunseo-Lab/glance/tree/{commit}",
    "built_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "platform": f"{os_name}/{arch}",
    "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
    "go": run("go", "version"),
    "node": run("node", "--version"),
    "scope": "reviewable personal release candidate; no Mac or public HTTPS cutover claimed",
}
(root / "release.json").write_text(json.dumps(manifest, indent=2) + "\n")
checksums = []
for file in sorted(root.iterdir()):
    if file.is_file() and file.name != "SHA256SUMS":
        checksums.append(
            hashlib.sha256(file.read_bytes()).hexdigest() + "  " + file.name
        )
(root / "SHA256SUMS").write_text("\n".join(checksums) + "\n")
print(json.dumps(manifest))
