FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /connect ./examples/aws/connect
ENTRYPOINT ["/connect"]
