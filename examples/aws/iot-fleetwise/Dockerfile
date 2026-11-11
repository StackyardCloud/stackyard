FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-fleetwise ./examples/aws/iot-fleetwise
ENTRYPOINT ["/iot-fleetwise"]
