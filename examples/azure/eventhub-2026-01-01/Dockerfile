FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /eventhub-2026-01-01-example ./examples/azure/eventhub-2026-01-01

ENTRYPOINT ["/eventhub-2026-01-01-example"]
