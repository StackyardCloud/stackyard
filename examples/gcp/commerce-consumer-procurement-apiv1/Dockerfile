FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /commerce-consumer-procurement-apiv1 ./examples/gcp/commerce-consumer-procurement-apiv1

ENTRYPOINT ["/commerce-consumer-procurement-apiv1"]
