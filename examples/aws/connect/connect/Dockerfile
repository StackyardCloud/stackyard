FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /connect ./examples/aws/connect/connect
ENTRYPOINT ["/connect"]
