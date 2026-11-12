FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /managedkafka-schemaregistry-apiv1 ./examples/gcp/managedkafka-schemaregistry-apiv1

ENTRYPOINT ["/managedkafka-schemaregistry-apiv1"]
