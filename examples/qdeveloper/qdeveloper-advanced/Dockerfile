FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qdeveloper-advanced ./examples/qdeveloper/qdeveloper-advanced
ENTRYPOINT ["/qdeveloper-advanced"]
