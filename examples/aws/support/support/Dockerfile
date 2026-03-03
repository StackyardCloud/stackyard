FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /support ./examples/aws/support/support
ENTRYPOINT ["/support"]
