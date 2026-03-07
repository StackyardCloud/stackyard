FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-fleetwise ./examples/aws/iot-fleetwise
ENTRYPOINT ["/iot-fleetwise"]
