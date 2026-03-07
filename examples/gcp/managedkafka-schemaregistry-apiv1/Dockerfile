FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /managedkafka-schemaregistry-apiv1 ./examples/gcp/managedkafka-schemaregistry-apiv1

ENTRYPOINT ["/managedkafka-schemaregistry-apiv1"]
