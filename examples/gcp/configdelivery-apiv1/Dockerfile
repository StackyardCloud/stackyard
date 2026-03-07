FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /configdelivery-apiv1 ./examples/gcp/configdelivery-apiv1

ENTRYPOINT ["/configdelivery-apiv1"]
