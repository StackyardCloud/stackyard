FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchlogs ./examples/aws/cloudwatchlogs/cloudwatchlogs
ENTRYPOINT ["/cloudwatchlogs"]
