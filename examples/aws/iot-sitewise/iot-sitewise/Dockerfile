FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-sitewise ./examples/aws/iot-sitewise/iot-sitewise
ENTRYPOINT ["/iot-sitewise"]
