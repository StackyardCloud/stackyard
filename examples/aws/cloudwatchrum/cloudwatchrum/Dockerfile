FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchrum ./examples/aws/cloudwatchrum/cloudwatchrum
ENTRYPOINT ["/cloudwatchrum"]
