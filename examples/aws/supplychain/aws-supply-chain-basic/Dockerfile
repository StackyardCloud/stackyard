FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-supply-chain-basic ./examples/aws/supplychain/aws-supply-chain-basic
ENTRYPOINT ["/aws-supply-chain-basic"]
