FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /transfer-advanced ./examples/aws/transfer/transfer-advanced
ENTRYPOINT ["/transfer-advanced"]
