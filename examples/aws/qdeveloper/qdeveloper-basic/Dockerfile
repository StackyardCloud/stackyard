FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qdeveloper-basic ./examples/aws/qdeveloper/qdeveloper-basic
ENTRYPOINT ["/qdeveloper-basic"]
