FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography ./examples/aws/payment-cryptography/payment-cryptography
ENTRYPOINT ["/payment-cryptography"]
