FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /datadog-2025-06-11-example ./examples/azure/datadog-2025-06-11

ENTRYPOINT ["/datadog-2025-06-11-example"]
