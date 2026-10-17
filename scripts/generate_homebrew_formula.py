#!/usr/bin/env python3
"""Generate a Homebrew tap formula from the release's verified archive hashes."""

import pathlib
import re
import sys


OWNER_REPO = "Agrim-P777/qubex"


def main() -> None:
    if len(sys.argv) != 2 or not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", sys.argv[1]):
        raise SystemExit("usage: generate_homebrew_formula.py VERSION (for example 0.1.0)")
    version = sys.argv[1]
    root = pathlib.Path("dist")
    hashes = {}
    for line in (root / "SHA256SUMS").read_text(encoding="utf-8").splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise SystemExit(f"invalid checksum for {name}")
        hashes[name] = digest

    lines = [
        "class Qbx < Formula",
        '  desc "Build jq queries by navigating JSON"',
        f'  homepage "https://github.com/{OWNER_REPO}"',
        f'  version "{version}"',
        '  license "MIT"',
        "",
    ]
    for target_os in ("macos", "linux"):
        archive_os = "darwin" if target_os == "macos" else "linux"
        lines.append(f"  on_{target_os} do")
        for selector, arch in (("arm", "arm64"), ("intel", "amd64")):
            name = f"qbx_{version}_{archive_os}_{arch}.tar.gz"
            if name not in hashes:
                raise SystemExit(f"missing checksum for {name}")
            lines.extend((
                f"    on_{selector} do",
                f'      url "https://github.com/{OWNER_REPO}/releases/download/v{version}/{name}"',
                f'      sha256 "{hashes[name]}"',
                "    end",
            ))
        lines.extend(("  end", ""))
    lines.extend((
        "  def install",
        '    bin.install "qbx"',
        '    pkgshare.install "THIRD_PARTY_NOTICES.txt"',
        "  end",
        "",
        "  test do",
        '    assert_match "Qubex #{version}", shell_output("#{bin}/qbx --version")',
        "  end",
        "end",
        "",
    ))
    output = root / "Formula" / "qbx.rb"
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text("\n".join(lines), encoding="utf-8")
    print(output)


if __name__ == "__main__":
    main()
