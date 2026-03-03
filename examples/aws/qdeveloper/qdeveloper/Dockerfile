FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qdeveloper ./examples/aws/qdeveloper/qdeveloper
ENTRYPOINT ["/qdeveloper"]
