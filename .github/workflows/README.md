# Distribution and Global Install

This project ships binaries through GitHub Releases via [release.yml](./release.yml) and [.goreleaser.yaml](../../.goreleaser.yaml).

## Release flow

From repository root:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The `release` workflow builds Linux/macOS binaries (`amd64`, `arm64`), publishes a GitHub Release, and uploads `checksums.txt`.

## Global installation

### Option 1: Install from source (Go)

```bash
go install github.com/TeoBale/dualgit/cmd/dualgit@latest
```

Make sure `$(go env GOPATH)/bin` (or your `GOBIN`) is in `PATH`.

### Option 2: Install from GitHub Release binary

Example for macOS arm64:

```bash
VERSION=v0.1.0
curl -L -o dualgit.tar.gz "https://github.com/TeoBale/dualgit/releases/download/${VERSION}/dualgit_${VERSION#v}_darwin_arm64.tar.gz"
tar -xzf dualgit.tar.gz
sudo install dualgit /usr/local/bin/dualgit
```

Then verify:

```bash
dualgit status
```
