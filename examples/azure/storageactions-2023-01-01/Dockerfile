FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /storageactions-2023-01-01 ./examples/azure/storageactions-2023-01-01

ENTRYPOINT ["/storageactions-2023-01-01"]
