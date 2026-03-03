FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /internet-monitor-basic ./examples/aws/internet-monitor/internet-monitor-basic
ENTRYPOINT ["/internet-monitor-basic"]
