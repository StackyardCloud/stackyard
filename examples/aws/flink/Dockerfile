FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /flink ./examples/aws/flink
ENTRYPOINT ["/flink"]
