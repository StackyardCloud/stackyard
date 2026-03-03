FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-greengrass-basic ./examples/aws/iot-greengrass/iot-greengrass-basic
ENTRYPOINT ["/iot-greengrass-basic"]
