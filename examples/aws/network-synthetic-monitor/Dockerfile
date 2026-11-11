FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-synthetic-monitor ./examples/aws/network-synthetic-monitor
ENTRYPOINT ["/network-synthetic-monitor"]
