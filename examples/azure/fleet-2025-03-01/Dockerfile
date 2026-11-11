FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /fleet-2025-03-01-example ./examples/azure/fleet-2025-03-01

ENTRYPOINT ["/fleet-2025-03-01-example"]
