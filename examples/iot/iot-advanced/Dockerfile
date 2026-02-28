FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-advanced ./examples/iot/iot-advanced
ENTRYPOINT ["/iot-advanced"]
