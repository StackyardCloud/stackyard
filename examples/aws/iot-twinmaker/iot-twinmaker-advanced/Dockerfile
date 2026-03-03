FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-twinmaker-advanced ./examples/aws/iot-twinmaker/iot-twinmaker-advanced
ENTRYPOINT ["/iot-twinmaker-advanced"]
