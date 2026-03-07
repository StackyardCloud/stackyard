FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-mi ./examples/aws/iot-mi
ENTRYPOINT ["/iot-mi"]
