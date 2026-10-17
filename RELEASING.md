# Releases and Homebrew

Qubex builds four CGO-free binaries: macOS and Linux, each on amd64 and arm64.
The `Qubex release` GitHub Actions workflow publishes versioned archives, raw
binaries, SHA256SUMS, dependency notices, and a generated Homebrew formula when
you push a `vMAJOR.MINOR.PATCH` tag on a commit contained in `main`.

## Publish a GitHub Release

1. Merge and push the intended changes to `main`. Confirm CI is green and the
   working tree is clean. Update `CHANGELOG.md` for the version.
2. From the release commit, create and push an annotated tag:

   ```sh
   git switch main
   git pull --ff-only origin main
   git tag -a v0.1.0 -m "Qubex v0.1.0"
   git push origin v0.1.0
   ```

3. Watch **Actions → Qubex release**. Once it succeeds, check the
   [Releases page](https://github.com/Agrim-P777/qubex/releases) for all four
   `qbx_0.1.0_OS_ARCH.tar.gz` archives, four raw binaries, `SHA256SUMS`,
   `THIRD_PARTY_NOTICES.txt`, and `qbx.rb`.

The tag must point to a commit already on `main`; the workflow builds the tag's
code. A prerelease tag such as `v0.2.0-rc.1` is marked as a prerelease. Never
move a published version tag: use a new version for fixes.

To inspect packages locally before tagging:

```sh
make release VERSION=0.1.0
python3 scripts/generate_homebrew_formula.py 0.1.0
cd dist && shasum -a 256 -c SHA256SUMS
```

## Publish the Homebrew tap

The first time, create a **public** GitHub repository named
`Agrim-P777/homebrew-tap`. Put the release-generated `qbx.rb` at
`Formula/qbx.rb` in that repository and push it. From a clone of that tap,
download the formula attached to the matching Qubex release:

```sh
mkdir -p Formula
gh release download v0.1.0 --repo Agrim-P777/qubex --pattern qbx.rb --dir Formula
git add Formula/qbx.rb
git commit -m "Add qbx 0.1.0"
git push origin main
```

For each later release, copy the newly generated formula from the matching
release into the tap and commit it. The formula's URLs and checksums are tied
to that exact release; do not edit them by hand. Check the formula with
`brew style Formula/qbx.rb` and install it on a supported macOS/Linux machine.
The release workflow does **not** update a separate tap automatically.

Users can then run:

```sh
brew tap Agrim-P777/tap
brew trust --formula Agrim-P777/tap/qbx
brew install Agrim-P777/tap/qbx
```

After tapping and trusting, `brew install qbx` also works if no formula of that
name in another tap takes precedence. A one-command `brew install qbx` for
people who have never tapped Qubex requires acceptance into `homebrew/core`.
The separate tap is the immediate distribution route.

**Visibility:** GitHub Release assets in a private repository cannot be
downloaded by anonymous Homebrew users. Make the Qubex repository public
before advertising public installation, and keep the tap public too. A private
team-only tap/release needs authenticated access and is a different
installation flow.
