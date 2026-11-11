FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /data-manager-for-agri-resourcemanager-2023-06-01-preview ./examples/azure/data-manager-for-agri-resourcemanager-2023-06-01-preview

ENTRYPOINT ["/data-manager-for-agri-resourcemanager-2023-06-01-preview"]
