FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /pubsub-apiv1 ./examples/gcp/pubsub-apiv1

ENTRYPOINT ["/pubsub-apiv1"]
