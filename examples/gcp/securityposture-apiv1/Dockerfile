FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /securityposture-apiv1 ./examples/gcp/securityposture-apiv1

ENTRYPOINT ["/securityposture-apiv1"]
