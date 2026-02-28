FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iot-twinmaker-basic ./examples/iot-twinmaker/iot-twinmaker-basic
ENTRYPOINT ["/iot-twinmaker-basic"]
