FROM golang:1.15.2 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY ./ ./

RUN GOOS=linux CGO_ENABLED=0 go build

FROM alpine:3.12.0

COPY --from=builder /app/rv-homekit /bin/rv-homekit

CMD ["/bin/rv-homekit","-configFile=/var/lib/rv-homekit/config.json"]