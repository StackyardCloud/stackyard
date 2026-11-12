FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /data-plane-2024-02-01-preview ./examples/azure/api-center/data-plane-2024-02-01-preview

ENTRYPOINT ["/data-plane-2024-02-01-preview"]
