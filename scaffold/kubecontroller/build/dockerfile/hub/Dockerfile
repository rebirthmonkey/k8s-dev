# Build the manager binary
FROM golang:1.18 as builder
#RUN apt-get -y update && apt-get -y install upx

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod tidy
RUN go mod download

# Copy the go source
COPY cmd/hub/main.go main.go
COPY apis apis/
COPY pkg pkg/
COPY configs configs/

# Build
#ARG VERSION
#ARG COMMIT
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
#    -ldflags="-s -w -X 'hub.docker.com/rebirthmonkey/k8s-dev/pkg/version.Version=${VERSION}' -X 'hub.docker.com/rebirthmonkey/k8s-dev/pkg/version.Commit=${COMMIT}'" \
#    -o hub main.go && \
#    upx --brute -9 hub

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -o hub main.go


# Use alpine as minimal base image to package the manager binary
FROM alpine:3.13.6
RUN apk update \
       && apk upgrade \
       && apk add --no-cache bash curl busybox-extras tree \
       bash-doc \
       bash-completion \
       libc6-compat \
       && rm -rf /var/cache/apk/* \
       && /bin/bash

COPY configs/kubecontroller.yaml /opt/kubecontroller/configs/

RUN chmod -R 777 /opt/kubecontroller/*
RUN mkdir /opt/kubecontroller/logs \
    && mkdir /opt/kubecontroller/workspace \
    && chmod -R 777 /opt/kubecontroller/logs/ \
    && chmod -R 777 /opt/kubecontroller/workspace/

ENV TF_LOG=WARN

WORKDIR /
COPY --from=builder /workspace/hub /opt/kubecontroller/bin/hub
USER 0:0

ENTRYPOINT ["/opt/kubecontroller/bin/hub"]
CMD ["-c", "/opt/kubecontroller/configs/kubecontroller.yaml"]