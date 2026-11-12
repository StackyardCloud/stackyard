FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /elasticsan-2025-09-01-example ./examples/azure/elasticsan-2025-09-01

ENTRYPOINT ["/elasticsan-2025-09-01-example"]
