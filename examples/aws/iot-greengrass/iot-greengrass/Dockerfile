FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-greengrass ./examples/aws/iot-greengrass/iot-greengrass
ENTRYPOINT ["/iot-greengrass"]
