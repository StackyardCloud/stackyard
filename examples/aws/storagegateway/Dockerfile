FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /storagegateway ./examples/aws/storagegateway
ENTRYPOINT ["/storagegateway"]
