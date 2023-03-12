# Build the manager binary
FROM golang:1.18 as builder
#RUN apt-get -y update && apt-get -y install upx

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer


# Copy the go source
COPY cmd/hub/main.go main.go
COPY apis apis/
COPY pkg pkg/
COPY configs configs/

RUN go mod tidy
RUN go mod download

# Build
#ARG VERSION
#ARG COMMIT
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
#    -ldflags="-s -w -X 'hub.docker.com/rebirthmonkey/k8s-dev/pkg/version.Version=${VERSION}' -X 'hub.docker.com/rebirthmonkey/k8s-dev/pkg/version.Commit=${COMMIT}'" \
#    -o hub main.go && \
#    upx --brute -9 hub

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go


FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY configs/kubecontroller.yaml .
COPY configs/cert /Users/ruan/workspace/k8s-dev/scaffold/kubecontroller/configs/cert
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
CMD ["-c", "/kubecontroller.yaml"]