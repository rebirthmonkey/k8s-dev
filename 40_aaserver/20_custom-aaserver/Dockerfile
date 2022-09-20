FROM golang:1.18 as build
LABEL maintainer="<rebirthmonkey@gmail.com>"

WORKDIR /opt/aaserver

COPY . /opt/aaserver
RUN CGO_ENABLED=0 GOOS=linux go build main.go


FROM alpine:latest
RUN apk --no-cache add ca-certificates

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
      echo "Asia/Shanghai" > /etc/timezone

WORKDIR /opt/aaserver/bin
COPY --from=build /opt/aaserver/main /opt/aaserver/bin

WORKDIR /etc/aaserver/cert
COPY configs/cert /etc/aaserver

ENTRYPOINT ["/opt/aaserver/bin/main"]
CMD ["--client-ca-file", "/etc/aaserver/cert/ca.crt"]
