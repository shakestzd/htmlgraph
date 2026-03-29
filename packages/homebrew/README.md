# Homebrew Tap for HtmlGraph

Homebrew formula for [htmlgraph](https://github.com/shakestzd/htmlgraph) — local-first observability and coordination platform for AI-assisted development.

## Setup (one-time)

1. Create the GitHub tap repo: `shakestzd/homebrew-htmlgraph`
2. Copy `htmlgraph.rb` to that repo

## Usage

```bash
brew tap shakestzd/htmlgraph
brew install htmlgraph
```

## Updating

After a new release, run the update script from this directory:

```bash
./update-formula.sh 0.36.0
```

This will:
- Download the checksums file for the specified version from GitHub Releases
- Parse SHA256 values for all four platforms (darwin/linux x amd64/arm64)
- Update `htmlgraph.rb` in-place with the new version and correct checksums

Then commit and push the updated formula to the tap repo:

```bash
git add htmlgraph.rb
git commit -m "htmlgraph 0.36.0"
git push
```

## Formula Details

The formula installs pre-built binaries from GitHub Releases. No compilation required.

Supported platforms:
- macOS arm64 (Apple Silicon)
- macOS amd64 (Intel)
- Linux arm64
- Linux amd64

Release asset URL pattern:
```
https://github.com/shakestzd/htmlgraph/releases/download/go/v{VERSION}/htmlgraph_{VERSION}_{OS}_{ARCH}.tar.gz
```
