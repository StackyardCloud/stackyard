FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography ./examples/aws/payment-cryptography
ENTRYPOINT ["/payment-cryptography"]
