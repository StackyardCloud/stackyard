FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchrum-advanced ./examples/aws/cloudwatchrum/cloudwatchrum-advanced
ENTRYPOINT ["/cloudwatchrum-advanced"]
