FROM golang:1.17 as builder
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o home-alerter

FROM alpine:3.15 as additionals
RUN apk add tzdata ca-certificates
RUN update-ca-certificates

# final stage
FROM scratch
COPY --from=builder /app/home-alerter /app/
COPY --from=additionals /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=additionals /usr/share/zoneinfo /usr/share/zoneinfo
ENTRYPOINT ["/app/home-alerter"]