FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /edge-actions-2025-12-01-preview ./examples/azure/edge-actions-2025-12-01-preview

ENTRYPOINT ["/edge-actions-2025-12-01-preview"]
