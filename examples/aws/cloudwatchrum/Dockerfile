FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchrum ./examples/aws/cloudwatchrum
ENTRYPOINT ["/cloudwatchrum"]
