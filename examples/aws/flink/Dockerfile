FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /flink ./examples/aws/flink
ENTRYPOINT ["/flink"]
