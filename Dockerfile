FROM golang:1.14 AS builder

WORKDIR /go/src/github.com/rrreeeyyy/prometheus-rds-sd

COPY . .

RUN env GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o /prometheus-rds-sd .

FROM alpine:edge
RUN apk add --update --no-cache ca-certificates
COPY --from=builder /prometheus-rds-sd /prometheus-rds-sd
USER nobody
ENTRYPOINT ["/prometheus-rds-sd"]
