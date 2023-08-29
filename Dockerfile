FROM golang:1.21 AS builder
WORKDIR /src

# This arg is passed by docker buildx & contains the target CPU architecture (e.g., amd64, arm64, etc.)
ARG TARGETARCH
ARG TARGETOS

ENV GOARCH=$TARGETARCH
ENV GOOS=$TARGETOS

ENV CGO_ENABLED=0

COPY . .

RUN go build -o dockerless main.go

# use musl busybox since it's staticly compiled on all platforms
FROM busybox:musl AS busybox

# now build from scratch
FROM scratch

# Create kaniko directory with world write permission to allow non root run
RUN --mount=from=busybox,dst=/usr/ ["busybox", "sh", "-c", "mkdir -p /.dockerless && chmod 777 /.dockerless"]

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /.dockerless/ssl/certs/
COPY files/nsswitch.conf /etc/nsswitch.conf
ENV HOME /root
ENV USER root
ENV KANIKO_DIR /.dockerless
ENV PATH /usr/local/bin:/.dockerless:/.dockerless/bin
ENV SSL_CERT_DIR=/.dockerless/ssl/certs
ENV DOCKER_CONFIG /.dockerless/.docker/

COPY --from=builder /src/dockerless /.dockerless/dockerless
COPY --from=busybox /bin /.dockerless/bin

WORKDIR /.dockerless

ENTRYPOINT ["/.dockerless/bin/sh", "-c"]
CMD ["sleep infinity"]