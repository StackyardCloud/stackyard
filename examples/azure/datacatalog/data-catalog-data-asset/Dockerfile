FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /data-catalog-data-asset ./examples/azure/datacatalog/data-catalog-data-asset

ENTRYPOINT ["/data-catalog-data-asset"]
