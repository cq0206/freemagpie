"""Repackage existing, verified macOS binaries with the distribution name."""
import pathlib
import plistlib
import subprocess
import sys
import tempfile

dist = pathlib.Path(sys.argv[1]).resolve()
for arch in ("arm64", "amd64"):
    with tempfile.TemporaryDirectory() as directory:
        root = pathlib.Path(directory)
        subprocess.run(["ditto", "-x", "-k", str(dist / f"magpie-darwin-{arch}.zip"), str(root)], check=True)
        app = root / "FreeMagpie.app"
        (root / "magpie.app").rename(app)
        plist = app / "Contents/Info.plist"
        info = plistlib.loads(plist.read_bytes())
        info.update(CFBundleName="FreeMagpie", CFBundleDisplayName="FreeMagpie")
        plist.write_bytes(plistlib.dumps(info))
        subprocess.run(["codesign", "--force", "--deep", "--sign", "-", str(app)], check=True)
        subprocess.run(["codesign", "--verify", "--deep", "--strict", str(app)], check=True)
        archive = dist / f"freemagpie-darwin-{arch}.zip"
        archive.unlink(missing_ok=True)
        subprocess.run(["ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", str(app), str(archive)], check=True)
