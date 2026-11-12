FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /internet-monitor ./examples/aws/internet-monitor
ENTRYPOINT ["/internet-monitor"]
