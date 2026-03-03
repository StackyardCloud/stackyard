FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-sitewise-advanced ./examples/aws/iot-sitewise/iot-sitewise-advanced
ENTRYPOINT ["/iot-sitewise-advanced"]
