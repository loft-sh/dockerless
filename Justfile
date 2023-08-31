set positional-arguments

[private]
alias align := check-structalign

_default:
  @just --list

# Run golangci-lint for all packages
lint *ARGS:
  golangci-lint run {{ARGS}}

# --- Build ---

# Build the loft docker image for a given tag
build-image tag="ghcr.io/loft-sh/dockerless:release-test":
  GGOOS=linux just build

  cp dist/loft_dockerless_$(go env GOARCH | sed 's/amd64/amd64_v1/g')/loft ./loft

  docker build -t {{tag}} -f Dockerfile --build-arg TARGETARCH=$(uname -m) --build-arg TARGETOS=linux .

  rm ./dockerless

# Delete the e2e docker image
delete-image:
  docker rmi ghcr.io/loft-sh/dockerless:release-test

# --- Build ---

# Build the binary given its id
build id="": _download-goreleaser-nightly
  SNAPSHOT_VERSION=$(git describe --tags `git rev-list --tags --max-count=1` | sed 's/v//g' ) \
    LICENSE_PUBLIC_KEY="" \
    LICENSE_SERVER_ENABLED=true \
    LICENSE_SERVER=https://license.dev.loft.sh \
    ./tools/goreleaser/goreleaser build --snapshot --id {{id}} --clean

# Download goreleaser nightly
_download-goreleaser-nightly:
  #!/bin/sh
  set -e

  FOLDER="tools/goreleaser"
  TAR_FILE="tools/goreleaser.tar.gz"

  rm -f "$TAR_FILE"
  rm -rf "$FOLDER"

  mkdir -p "$FOLDER"

  curl -s -L -o "$TAR_FILE" \
    "https://github.com/goreleaser/goreleaser/releases/download/nightly/goreleaser_$(uname -s)_$(uname -m).tar.gz"

  tar -xf "$TAR_FILE" -C "$FOLDER"
  rm "$TAR_FILE"

  chmod +x "$FOLDER/goreleaser"

# Clean the release folder
[private]
clean-release:
  rm -rf ./release

# Check struct memory alignment and print potential improvements
[no-exit-message]
check-structalign flags="":
  go run github.com/dkorunic/betteralign/cmd/betteralign@latest {{flags}} ./...
