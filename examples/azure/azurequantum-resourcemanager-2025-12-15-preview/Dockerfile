FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azurequantum-resourcemanager-2025-12-15-preview ./examples/azure/azurequantum-resourcemanager-2025-12-15-preview

ENTRYPOINT ["/azurequantum-resourcemanager-2025-12-15-preview"]
