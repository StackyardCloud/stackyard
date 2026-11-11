FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-wireless ./examples/aws/iot-wireless
ENTRYPOINT ["/iot-wireless"]
