FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /transfer-basic ./examples/aws/transfer/transfer-basic
ENTRYPOINT ["/transfer-basic"]
