FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iotevents-advanced ./examples/aws/iotevents/iotevents-advanced
ENTRYPOINT ["/iotevents-advanced"]
