FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /networkmanagement-apiv1 ./examples/gcp/networkmanagement-apiv1

ENTRYPOINT ["/networkmanagement-apiv1"]
