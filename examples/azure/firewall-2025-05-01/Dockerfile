FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /firewall-2025-05-01-example ./examples/azure/firewall-2025-05-01

ENTRYPOINT ["/firewall-2025-05-01-example"]
