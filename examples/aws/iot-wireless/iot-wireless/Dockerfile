FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-wireless ./examples/aws/iot-wireless/iot-wireless
ENTRYPOINT ["/iot-wireless"]
