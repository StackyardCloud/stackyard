FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /kms-inventory-apiv1 ./examples/gcp/kms-inventory-apiv1

ENTRYPOINT ["/kms-inventory-apiv1"]

