FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /impact-2026-01-01-preview ./examples/azure/impact-2026-01-01-preview

ENTRYPOINT ["/impact-2026-01-01-preview"]
