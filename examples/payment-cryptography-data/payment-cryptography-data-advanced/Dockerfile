FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography-data-advanced ./examples/payment-cryptography-data/payment-cryptography-data-advanced
ENTRYPOINT ["/payment-cryptography-data-advanced"]
