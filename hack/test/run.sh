#!/usr/bin/env bash

set -e

# Build image
docker build -t dockerless .

# Test Image
docker run --rm --name dockerless -v "$(pwd)/hack/test:/workspaces/test" dockerless "/.dockerless/dockerless --context dir:///workspaces/test --dockerfile /workspaces/test/Dockerfile && sleep infinity"