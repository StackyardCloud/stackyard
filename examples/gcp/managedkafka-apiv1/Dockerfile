FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /managedkafka-apiv1 ./examples/gcp/managedkafka-apiv1

ENTRYPOINT ["/managedkafka-apiv1"]
