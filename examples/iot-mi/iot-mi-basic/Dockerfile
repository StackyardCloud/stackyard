FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-mi-basic ./examples/iot-mi/iot-mi-basic
ENTRYPOINT ["/iot-mi-basic"]
