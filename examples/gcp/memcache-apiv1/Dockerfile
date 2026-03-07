FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /memcache-apiv1 ./examples/gcp/memcache-apiv1

ENTRYPOINT ["/memcache-apiv1"]
