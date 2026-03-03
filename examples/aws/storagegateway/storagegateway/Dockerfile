FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /storagegateway ./examples/aws/storagegateway/storagegateway
ENTRYPOINT ["/storagegateway"]
