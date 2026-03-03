FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /firehose-basic ./examples/aws/firehose/firehose-basic
ENTRYPOINT ["/firehose-basic"]
