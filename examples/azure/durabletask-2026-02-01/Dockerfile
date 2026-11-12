FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /durabletask-2026-02-01-example ./examples/azure/durabletask-2026-02-01

ENTRYPOINT ["/durabletask-2026-02-01-example"]
