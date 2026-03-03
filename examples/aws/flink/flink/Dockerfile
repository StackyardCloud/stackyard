FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /flink ./examples/aws/flink/flink
ENTRYPOINT ["/flink"]
