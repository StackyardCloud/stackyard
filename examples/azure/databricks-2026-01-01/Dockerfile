FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /databricks-2026-01-01-example ./examples/azure/databricks-2026-01-01

ENTRYPOINT ["/databricks-2026-01-01-example"]
