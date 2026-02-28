FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-wireless-advanced ./examples/iot-wireless/iot-wireless-advanced
ENTRYPOINT ["/iot-wireless-advanced"]
