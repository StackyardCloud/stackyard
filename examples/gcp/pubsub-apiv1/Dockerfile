FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /pubsub-apiv1 ./examples/gcp/pubsub-apiv1

ENTRYPOINT ["/pubsub-apiv1"]
