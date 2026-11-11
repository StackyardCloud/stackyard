FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dataprotection-2026-03-01-example ./examples/azure/dataprotection-2026-03-01

ENTRYPOINT ["/dataprotection-2026-03-01-example"]
