FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /databasewatcher-2025-01-02-example ./examples/azure/databasewatcher-2025-01-02

ENTRYPOINT ["/databasewatcher-2025-01-02-example"]
