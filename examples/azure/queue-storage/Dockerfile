FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azure-queue-storage ./examples/azure/queue-storage

ENTRYPOINT ["/azure-queue-storage"]
