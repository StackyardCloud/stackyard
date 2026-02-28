FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iotevents-advanced ./examples/iotevents/iotevents-advanced
ENTRYPOINT ["/iotevents-advanced"]
