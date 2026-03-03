FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /storagegateway-advanced ./examples/aws/storagegateway/storagegateway-advanced
ENTRYPOINT ["/storagegateway-advanced"]
