FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /firehose ./examples/aws/firehose
ENTRYPOINT ["/firehose"]
