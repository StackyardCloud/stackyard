FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /netapp-2025-12-01 ./examples/azure/netapp-2025-12-01

ENTRYPOINT ["/netapp-2025-12-01"]
