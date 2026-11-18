"""FreeMagpie downloads and legacy updaters must both have hashed assets."""
import hashlib
import json
import pathlib
import subprocess
import sys
import tempfile
import unittest


class ManifestTest(unittest.TestCase):
    def test_branded_downloads_and_legacy_update_assets(self):
        with tempfile.TemporaryDirectory() as directory:
            dist = pathlib.Path(directory)
            names = [f"magpie-darwin-{a}.zip" for a in ("arm64", "amd64")]
            names += [f"magpie-{os}-{a}{ext}" for os, ext in (("linux", ""), ("windows", ".exe")) for a in ("arm64", "amd64")]
            names += [f"magpie-cli-{os}-{a}{ext}" for os, ext in (("darwin", ""), ("linux", ""), ("windows", ".exe")) for a in ("arm64", "amd64")]
            names += ["magpie-cli-android-arm64"]
            names += ["free" + name for name in names]
            names += ["LICENSE", "THIRD_PARTY_NOTICES"]
            for name in names:
                (dist / name).write_bytes(name.encode())
            script = pathlib.Path(__file__).with_name("manifest.py")
            subprocess.run([sys.executable, str(script), "0.1.0", directory], check=True, capture_output=True)
            release = json.loads((dist / "latest.json").read_text())
            self.assertEqual(set(release["assets"]), set(names))
            for name, asset in release["assets"].items():
                self.assertTrue(asset["url"].endswith("/" + name))
                self.assertEqual(asset["sha256"], hashlib.sha256((dist / name).read_bytes()).hexdigest())
            (dist / "freemagpie-darwin-arm64.zip").unlink()
            failed = subprocess.run([sys.executable, str(script), "0.1.0", directory], capture_output=True, text=True)
            self.assertNotEqual(failed.returncode, 0)
            self.assertIn("missing/empty release asset", failed.stderr)


if __name__ == "__main__":
    unittest.main()
