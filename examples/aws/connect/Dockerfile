FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /connect ./examples/aws/connect
ENTRYPOINT ["/connect"]
