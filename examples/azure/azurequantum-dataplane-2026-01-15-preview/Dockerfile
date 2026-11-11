FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azurequantum-dataplane-2026-01-15-preview ./examples/azure/azurequantum-dataplane-2026-01-15-preview

ENTRYPOINT ["/azurequantum-dataplane-2026-01-15-preview"]
