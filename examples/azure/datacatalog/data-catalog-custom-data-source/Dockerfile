FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /data-catalog-custom-data-source ./examples/azure/datacatalog/data-catalog-custom-data-source

ENTRYPOINT ["/data-catalog-custom-data-source"]
