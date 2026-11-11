FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /blockchain ./examples/aws/blockchain
ENTRYPOINT ["/blockchain"]
