FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-greengrass-advanced ./examples/iot-greengrass/iot-greengrass-advanced
ENTRYPOINT ["/iot-greengrass-advanced"]
