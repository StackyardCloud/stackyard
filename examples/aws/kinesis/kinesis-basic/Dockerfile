FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kinesis-basic ./examples/aws/kinesis/kinesis-basic
ENTRYPOINT ["/kinesis-basic"]
