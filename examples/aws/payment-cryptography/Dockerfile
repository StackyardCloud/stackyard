FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography ./examples/aws/payment-cryptography
ENTRYPOINT ["/payment-cryptography"]
