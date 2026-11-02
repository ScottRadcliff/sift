import hashlib
from pathlib import Path
import tarfile
import tempfile
import unittest
import zipfile

from package_release import validate_version, write_archive, write_checksums


class ReleasePackagingTests(unittest.TestCase):
    def test_archive_contents_and_executable_permissions(self):
        entries = [("sift", b"binary bytes", 0o755), ("skills/sift/SKILL.md", b"skill instructions", 0o644)]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            tar = root / "release.tar.gz"
            write_archive(tar, entries)
            with tarfile.open(tar) as archive:
                self.assertEqual(archive.getnames(), [entry[0] for entry in entries])
                self.assertEqual(archive.getmember("sift").mode, 0o755)
                self.assertEqual(archive.extractfile("sift").read(), b"binary bytes")
            zip_path = root / "release.zip"
            write_archive(zip_path, entries)
            with zipfile.ZipFile(zip_path) as archive:
                self.assertEqual(archive.read("skills/sift/SKILL.md"), b"skill instructions")
                self.assertEqual((archive.getinfo("sift").external_attr >> 16) & 0o777, 0o755)

    def test_checksums_cover_archives_and_are_relative(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / "release.zip"
            archive.write_bytes(b"package")
            write_checksums(root, [archive])
            digest, name = (root / "checksums.txt").read_text().strip().split()
            self.assertEqual(name, "release.zip")
            self.assertEqual(digest, hashlib.sha256(archive.read_bytes()).hexdigest())
            with self.assertRaises(FileExistsError):
                write_checksums(root, [archive])

    def test_release_versions(self):
        for version in ["v1.0.0", "v0.1.2", "v1.0.0-rc.1"]:
            validate_version(version)
        for version in ["dev", "1.0.0", "v1", "v01.0.0", "v1.0.0-01", "v1.0.0/other", "v1.0.0;echo bad"]:
            with self.assertRaises(ValueError, msg=version):
                validate_version(version)

    def test_archives_are_repeatable_and_cannot_be_overwritten(self):
        with tempfile.TemporaryDirectory() as directory:
            for extension in ["tar.gz", "zip"]:
                first = Path(directory) / f"first.{extension}"
                second = Path(directory) / f"second.{extension}"
                entries = [("sift", b"binary", 0o755)]
                write_archive(first, entries)
                write_archive(second, entries)
                self.assertEqual(first.read_bytes(), second.read_bytes())
                with self.assertRaises(FileExistsError):
                    write_archive(first, entries)


if __name__ == "__main__":
    unittest.main()
