FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-fleetwise ./examples/aws/iot-fleetwise/iot-fleetwise
ENTRYPOINT ["/iot-fleetwise"]
