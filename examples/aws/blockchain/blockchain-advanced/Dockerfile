FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /blockchain-advanced ./examples/aws/blockchain/blockchain-advanced
ENTRYPOINT ["/blockchain-advanced"]
