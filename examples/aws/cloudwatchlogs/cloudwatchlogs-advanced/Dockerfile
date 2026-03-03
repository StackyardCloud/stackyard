FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchlogs-advanced ./examples/aws/cloudwatchlogs/cloudwatchlogs-advanced
ENTRYPOINT ["/cloudwatchlogs-advanced"]
