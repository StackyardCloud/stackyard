FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /blockchain ./examples/aws/blockchain/blockchain
ENTRYPOINT ["/blockchain"]
