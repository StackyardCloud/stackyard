FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /edgehardwarecenter-2024-02-01-example ./examples/azure/edgehardwarecenter-2024-02-01

ENTRYPOINT ["/edgehardwarecenter-2024-02-01-example"]

