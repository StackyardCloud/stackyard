FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iotevents ./examples/aws/iotevents/iotevents
ENTRYPOINT ["/iotevents"]
