FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /memorydb ./examples/aws/memorydb
ENTRYPOINT ["/memorydb"]
