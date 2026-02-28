FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-mi-advanced ./examples/iot-mi/iot-mi-advanced
ENTRYPOINT ["/iot-mi-advanced"]
