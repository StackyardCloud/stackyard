FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /braket ./examples/aws/braket
ENTRYPOINT ["/braket"]
