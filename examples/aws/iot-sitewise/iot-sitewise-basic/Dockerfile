FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-sitewise-basic ./examples/aws/iot-sitewise/iot-sitewise-basic
ENTRYPOINT ["/iot-sitewise-basic"]
