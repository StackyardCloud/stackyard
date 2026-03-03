FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kinesis-advanced ./examples/aws/kinesis/kinesis-advanced
ENTRYPOINT ["/kinesis-advanced"]
