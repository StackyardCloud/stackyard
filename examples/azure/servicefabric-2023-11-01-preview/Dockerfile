FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /servicefabric-2023-11-01-preview ./examples/azure/servicefabric-2023-11-01-preview

ENTRYPOINT ["/servicefabric-2023-11-01-preview"]
