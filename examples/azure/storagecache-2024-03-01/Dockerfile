FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storagecache-2024-03-01 ./examples/azure/storagecache-2024-03-01

ENTRYPOINT ["/storagecache-2024-03-01"]
