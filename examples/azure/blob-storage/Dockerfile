FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azure-blob-storage ./examples/azure/blob-storage

ENTRYPOINT ["/azure-blob-storage"]
