FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-synthetic-monitor-basic ./examples/aws/network-synthetic-monitor/network-synthetic-monitor-basic
ENTRYPOINT ["/network-synthetic-monitor-basic"]
