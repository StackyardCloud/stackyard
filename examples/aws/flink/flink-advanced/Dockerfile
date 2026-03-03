FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /flink-advanced ./examples/aws/flink/flink-advanced
ENTRYPOINT ["/flink-advanced"]
