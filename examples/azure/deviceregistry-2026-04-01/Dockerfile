FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /deviceregistry-2026-04-01-example ./examples/azure/deviceregistry-2026-04-01

ENTRYPOINT ["/deviceregistry-2026-04-01-example"]
