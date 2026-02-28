FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /flink-basic ./examples/flink/flink-basic
ENTRYPOINT ["/flink-basic"]
