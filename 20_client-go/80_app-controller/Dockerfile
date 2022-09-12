FROM golang:1.18 as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o app-controller cmd/app-controller.go

FROM alpine:3.15.3

WORKDIR /app

COPY --from=builder /app/app-controller .

CMD ["./app-controller"]