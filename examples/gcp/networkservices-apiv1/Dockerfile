FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /networkservices-apiv1 ./examples/gcp/networkservices-apiv1

ENTRYPOINT ["/networkservices-apiv1"]
