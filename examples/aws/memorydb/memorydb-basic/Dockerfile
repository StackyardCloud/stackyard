FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /memorydb-basic ./examples/aws/memorydb/memorydb-basic
ENTRYPOINT ["/memorydb-basic"]
