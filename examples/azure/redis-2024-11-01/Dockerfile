FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /redis-2024-11-01 ./examples/azure/redis-2024-11-01

ENTRYPOINT ["/redis-2024-11-01"]
