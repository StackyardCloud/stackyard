FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /internet-monitor-advanced ./examples/aws/internet-monitor/internet-monitor-advanced
ENTRYPOINT ["/internet-monitor-advanced"]
