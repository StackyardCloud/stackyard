FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /iotevents ./examples/aws/iotevents
ENTRYPOINT ["/iotevents"]
