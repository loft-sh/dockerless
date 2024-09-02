# dockerless

Dockerless is a tool that allows you to build and run Docker containers without having Docker installed.

## Features

- Build Docker images without Docker
- Run Docker containers without Docker

## Installation

WARN: running dockerless on your host can delete key files and destroy your working environment.

For example, to build an image, run the following in the gcr.io/kaniko-project/executor image:

``` bash
dockerless build --dockerfile Dockerfile --context .
```

And to run a container:

``` bash
dockerless start
```

## Development

Build dockerless and new image

```bash
just build
cp dist/dockerless_{arch}/dockerless .
docker build -t {repo}/dockerless:{tag} -f Dockerfile --build-arg TARGETARCH=$(uname -m) --build-arg TARGETOS=linux .
```