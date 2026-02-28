FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-wireless-basic ./examples/iot-wireless/iot-wireless-basic
ENTRYPOINT ["/iot-wireless-basic"]
