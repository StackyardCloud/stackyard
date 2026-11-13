FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /cosmosdb-2024-11-15 ./examples/azure/cosmosdb-2024-11-15

ENTRYPOINT ["/cosmosdb-2024-11-15"]
