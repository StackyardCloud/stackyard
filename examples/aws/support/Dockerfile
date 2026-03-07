FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /support ./examples/aws/support
ENTRYPOINT ["/support"]
