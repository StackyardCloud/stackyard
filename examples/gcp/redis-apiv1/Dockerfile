FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /redis-apiv1 ./examples/gcp/redis-apiv1

ENTRYPOINT ["/redis-apiv1"]
