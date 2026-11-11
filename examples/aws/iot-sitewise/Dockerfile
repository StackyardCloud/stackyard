FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-sitewise ./examples/aws/iot-sitewise
ENTRYPOINT ["/iot-sitewise"]
