FROM golang:1.19.1 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY ./ ./

RUN go test ./...

RUN GOOS=linux CGO_ENABLED=0 go build

FROM alpine:3.16.2

COPY --from=builder /app/rv-homekit /bin/rv-homekit

CMD ["/bin/rv-homekit","-configFile=/var/lib/rv-homekit/config.json"]
