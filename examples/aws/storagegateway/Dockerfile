FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /storagegateway ./examples/aws/storagegateway
ENTRYPOINT ["/storagegateway"]
