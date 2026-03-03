FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /payment-cryptography-basic ./examples/aws/payment-cryptography/payment-cryptography-basic
ENTRYPOINT ["/payment-cryptography-basic"]
