FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qdeveloper-advanced ./examples/aws/qdeveloper/qdeveloper-advanced
ENTRYPOINT ["/qdeveloper-advanced"]
