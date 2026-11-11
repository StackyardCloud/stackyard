FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /dellstorage-2025-03-21-example ./examples/azure/dellstorage-2025-03-21

ENTRYPOINT ["/dellstorage-2025-03-21-example"]

