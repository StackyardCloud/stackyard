FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /memorydb-advanced ./examples/aws/memorydb/memorydb-advanced
ENTRYPOINT ["/memorydb-advanced"]
