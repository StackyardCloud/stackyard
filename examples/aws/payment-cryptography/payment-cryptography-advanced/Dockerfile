FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography-advanced ./examples/aws/payment-cryptography/payment-cryptography-advanced
ENTRYPOINT ["/payment-cryptography-advanced"]
