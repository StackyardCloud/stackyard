FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /firehose-advanced ./examples/firehose/firehose-advanced
ENTRYPOINT ["/firehose-advanced"]
