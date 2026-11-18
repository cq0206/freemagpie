"""freemagpie: reject incomplete builds and emit updater + checksum metadata."""
import hashlib
import json
import pathlib
import re
import sys

version, directory = sys.argv[1:]
if not re.fullmatch(r"\d+\.\d+\.\d+", version):
    raise SystemExit("expected X.Y.Z")
dist = pathlib.Path(directory)
names = [f"magpie-darwin-{arch}.zip" for arch in ("arm64", "amd64")]
names += [f"magpie-windows-{arch}.exe" for arch in ("amd64", "arm64")]
names += [f"magpie-linux-{arch}" for arch in ("amd64", "arm64")]
names += [
    f"magpie-cli-{os}-{arch}" + (".exe" if os == "windows" else "")
    for os, arches in (
        ("darwin", ("arm64", "amd64")),
        ("linux", ("amd64", "arm64")),
        ("windows", ("amd64", "arm64")),
        ("android", ("arm64",)),
    )
    for arch in arches
]
names += ["free" + name for name in names]
names += ["LICENSE", "THIRD_PARTY_NOTICES"]
url = f"https://github.com/cq0206/freemagpie/releases/tag/freemagpie-v{version}"
download = url.replace("/tag/", "/download/")
assets = {}
for name in sorted(names):
    path = dist / name
    if not path.is_file() or path.stat().st_size == 0:
        raise SystemExit(f"missing/empty release asset: {path}")
    assets[name] = dict(
        url=f"{download}/{name}",
        size=path.stat().st_size,
        sha256=hashlib.file_digest(path.open("rb"), "sha256").hexdigest(),
    )
notes = (pathlib.Path(__file__).parent / "RELEASE_NOTES.md").read_text()
release = dict(version=version, notes=notes, url=url, assets=assets)
(dist / "latest.json").write_text(json.dumps(release, ensure_ascii=False, indent=2) + "\n")
(dist / "notes.json").write_text(json.dumps(dict(releases=[{k: release[k] for k in ("version", "notes", "url")}]), ensure_ascii=False, indent=2) + "\n")
checksums = {name: asset["sha256"] for name, asset in assets.items()}
for name in ("latest.json", "notes.json"):
    checksums[name] = hashlib.sha256((dist / name).read_bytes()).hexdigest()
(dist / "SHA256SUMS").write_text("".join(f"{checksum}  {name}\n" for name, checksum in sorted(checksums.items())))
print(f"validated {len(assets) - 2} binaries, licenses, metadata and checksums")
