FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-mi ./examples/aws/iot-mi/iot-mi
ENTRYPOINT ["/iot-mi"]
