FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-advanced ./examples/aws/iot/iot-advanced
ENTRYPOINT ["/iot-advanced"]
