FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /pubsublite-apiv1 ./examples/gcp/pubsublite-apiv1

ENTRYPOINT ["/pubsublite-apiv1"]
