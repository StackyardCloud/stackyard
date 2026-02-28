FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-fleetwise-advanced ./examples/iot-fleetwise/iot-fleetwise-advanced
ENTRYPOINT ["/iot-fleetwise-advanced"]
