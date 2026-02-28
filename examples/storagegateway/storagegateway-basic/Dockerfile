FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /storagegateway-basic ./examples/storagegateway/storagegateway-basic
ENTRYPOINT ["/storagegateway-basic"]
