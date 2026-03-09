FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /search-management-resource-manager-shared-usage-by-subscription-sku-2025-05-01 ./examples/azure/search-management/search-management-resource-manager-shared-usage-by-subscription-sku-2025-05-01

ENTRYPOINT ["/search-management-resource-manager-shared-usage-by-subscription-sku-2025-05-01"]
