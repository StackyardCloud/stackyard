FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iotevents-basic ./examples/aws/iotevents/iotevents-basic
ENTRYPOINT ["/iotevents-basic"]
