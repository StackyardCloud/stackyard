FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /internet-monitor ./examples/aws/internet-monitor/internet-monitor
ENTRYPOINT ["/internet-monitor"]
