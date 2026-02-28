FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchlogs-basic ./examples/cloudwatchlogs/cloudwatchlogs-basic
ENTRYPOINT ["/cloudwatchlogs-basic"]
