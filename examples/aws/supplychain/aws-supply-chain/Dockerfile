FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-supply-chain ./examples/aws/supplychain/aws-supply-chain
ENTRYPOINT ["/aws-supply-chain"]
