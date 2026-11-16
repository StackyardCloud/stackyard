FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /securityinsights-2025-09-01 ./examples/azure/securityinsights-2025-09-01

ENTRYPOINT ["/securityinsights-2025-09-01"]
