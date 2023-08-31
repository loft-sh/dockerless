FROM alpine as certs
RUN apk update && apk add ca-certificates

# use musl busybox since it's staticly compiled on all platforms
FROM busybox:musl AS busybox

# now build from scratch
FROM scratch

# Create kaniko directory with world write permission to allow non root run
RUN --mount=from=busybox,dst=/usr/ ["busybox", "sh", "-c", "mkdir -p /.dockerless && chmod 777 /.dockerless"]

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /.dockerless/ssl/certs/
COPY files/nsswitch.conf /etc/nsswitch.conf

ENV HOME /root
ENV USER root
ENV KANIKO_DIR /.dockerless
ENV PATH /usr/local/bin:/.dockerless:/.dockerless/bin
ENV SSL_CERT_DIR=/.dockerless/ssl/certs
ENV DOCKER_CONFIG /.dockerless/.docker/

COPY dockerless /.dockerless/dockerless
COPY --from=busybox /bin /.dockerless/bin

RUN ["/.dockerless/bin/sh", "-c", "echo 'root:x:0:0:root:/root:/.dockerless/bin/sh' > /etc/passwd && chmod 666 /etc/passwd"]

WORKDIR /.dockerless

ENTRYPOINT ["/.dockerless/bin/sh", "-c"]

CMD ["sleep infinity"]
