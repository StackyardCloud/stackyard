FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /devicestreaming-apiv1 ./examples/gcp/devicestreaming-apiv1

ENTRYPOINT ["/devicestreaming-apiv1"]
