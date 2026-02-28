FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /blockchain-basic ./examples/blockchain/blockchain-basic
ENTRYPOINT ["/blockchain-basic"]
