FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-synthetic-monitor-advanced ./examples/network-synthetic-monitor/network-synthetic-monitor-advanced
ENTRYPOINT ["/network-synthetic-monitor-advanced"]
