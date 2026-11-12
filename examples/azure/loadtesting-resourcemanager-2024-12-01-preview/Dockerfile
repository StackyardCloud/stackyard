FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /loadtesting-resourcemanager-2024-12-01-preview ./examples/azure/loadtesting-resourcemanager-2024-12-01-preview

ENTRYPOINT ["/loadtesting-resourcemanager-2024-12-01-preview"]
