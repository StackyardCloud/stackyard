FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /iot-apiv1 ./examples/gcp/iot-apiv1

ENTRYPOINT ["/iot-apiv1"]
