FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography-data ./examples/aws/payment-cryptography-data/payment-cryptography-data
ENTRYPOINT ["/payment-cryptography-data"]
