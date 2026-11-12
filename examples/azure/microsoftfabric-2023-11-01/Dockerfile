FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /microsoftfabric-2023-11-01-example ./examples/azure/microsoftfabric-2023-11-01

ENTRYPOINT ["/microsoftfabric-2023-11-01-example"]
