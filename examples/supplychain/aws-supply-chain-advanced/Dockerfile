FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-supply-chain-advanced ./examples/supplychain/aws-supply-chain-advanced
ENTRYPOINT ["/aws-supply-chain-advanced"]
