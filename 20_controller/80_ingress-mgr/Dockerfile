FROM golang:1.18 as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o ingress-mgr cmd/main.go

FROM alpine:3.15.3

WORKDIR /app

COPY --from=builder /app/ingress-mgr .

CMD ["./ingress-mgr"]