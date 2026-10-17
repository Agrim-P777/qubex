#!/usr/bin/env python3
"""Archive the four cross-built binaries and write checksums."""

import hashlib
import io
import pathlib
import sys
import tarfile


def main() -> None:
    version = sys.argv[1]
    root = pathlib.Path("dist")
    artifacts = []
    for target_os in ("darwin", "linux"):
        for target_arch in ("amd64", "arm64"):
            binary = root / f"qbx-{target_os}-{target_arch}"
            archive = root / f"qbx_{version}_{target_os}_{target_arch}.tar.gz"
            with tarfile.open(archive, "w:gz") as tar:
                for source, name, mode in (
                    (binary, "qbx", 0o755),
                    (pathlib.Path("LICENSE"), "LICENSE", 0o644),
                    (root / "THIRD_PARTY_NOTICES.txt", "THIRD_PARTY_NOTICES.txt", 0o644),
                ):
                    data = source.read_bytes()
                    info = tarfile.TarInfo(name)
                    info.size = len(data)
                    info.mode = mode
                    info.mtime = 0
                    tar.addfile(info, io.BytesIO(data))
            artifacts.extend((binary, archive))
    with (root / "SHA256SUMS").open("w", encoding="utf-8") as sums:
        for artifact in sorted(artifacts):
            sums.write(f"{hashlib.sha256(artifact.read_bytes()).hexdigest()}  {artifact.name}\n")


if __name__ == "__main__":
    main()
