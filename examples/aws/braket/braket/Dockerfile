FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /braket ./examples/aws/braket/braket
ENTRYPOINT ["/braket"]
