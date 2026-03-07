FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-sitewise ./examples/aws/iot-sitewise
ENTRYPOINT ["/iot-sitewise"]
