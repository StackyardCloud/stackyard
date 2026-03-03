FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatchrum-basic ./examples/aws/cloudwatchrum/cloudwatchrum-basic
ENTRYPOINT ["/cloudwatchrum-basic"]
