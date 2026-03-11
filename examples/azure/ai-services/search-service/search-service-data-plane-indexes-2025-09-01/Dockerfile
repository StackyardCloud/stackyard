FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /search-service-data-plane-indexes-2025-09-01 ./examples/azure/search-service/search-service-data-plane-indexes-2025-09-01

ENTRYPOINT ["/search-service-data-plane-indexes-2025-09-01"]
