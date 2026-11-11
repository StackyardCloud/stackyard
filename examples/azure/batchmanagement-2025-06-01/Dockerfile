FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /batchmanagement-2025-06-01 ./examples/azure/batchmanagement-2025-06-01

ENTRYPOINT ["/batchmanagement-2025-06-01"]
