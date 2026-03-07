FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /pubsub-v2-apiv1 ./examples/gcp/pubsub-v2-apiv1

ENTRYPOINT ["/pubsub-v2-apiv1"]
