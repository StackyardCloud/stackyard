FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-basic ./examples/iot/iot-basic
ENTRYPOINT ["/iot-basic"]
