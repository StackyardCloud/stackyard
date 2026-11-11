FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dns-dnsresolver-2025-05-01-example ./examples/azure/dns-dnsresolver-2025-05-01

ENTRYPOINT ["/dns-dnsresolver-2025-05-01-example"]
