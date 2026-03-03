FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-fleetwise-basic ./examples/aws/iot-fleetwise/iot-fleetwise-basic
ENTRYPOINT ["/iot-fleetwise-basic"]
