FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /redis-cluster-apiv1 ./examples/gcp/redis-cluster-apiv1

ENTRYPOINT ["/redis-cluster-apiv1"]
