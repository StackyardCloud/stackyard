FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /kms-apiv1 ./examples/gcp/kms-apiv1

ENTRYPOINT ["/kms-apiv1"]
