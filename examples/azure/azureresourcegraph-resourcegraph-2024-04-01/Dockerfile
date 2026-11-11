FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /azureresourcegraph-resourcegraph-2024-04-01 ./examples/azure/azureresourcegraph-resourcegraph-2024-04-01

ENTRYPOINT ["/azureresourcegraph-resourcegraph-2024-04-01"]
