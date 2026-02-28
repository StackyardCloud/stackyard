FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography-data-basic ./examples/payment-cryptography-data/payment-cryptography-data-basic
ENTRYPOINT ["/payment-cryptography-data-basic"]
