FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /kinesis ./examples/aws/kinesis
ENTRYPOINT ["/kinesis"]
