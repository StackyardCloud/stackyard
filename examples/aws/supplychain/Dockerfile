FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-supply-chain ./examples/aws/supplychain
ENTRYPOINT ["/aws-supply-chain"]
