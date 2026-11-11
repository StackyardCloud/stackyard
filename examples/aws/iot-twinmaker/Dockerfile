FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-twinmaker ./examples/aws/iot-twinmaker
ENTRYPOINT ["/iot-twinmaker"]
