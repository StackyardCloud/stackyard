FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-twinmaker ./examples/aws/iot-twinmaker/iot-twinmaker
ENTRYPOINT ["/iot-twinmaker"]
