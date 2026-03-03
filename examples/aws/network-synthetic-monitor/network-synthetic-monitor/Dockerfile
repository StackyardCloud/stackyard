FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-synthetic-monitor ./examples/aws/network-synthetic-monitor/network-synthetic-monitor
ENTRYPOINT ["/network-synthetic-monitor"]
