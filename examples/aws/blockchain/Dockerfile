FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /blockchain ./examples/aws/blockchain
ENTRYPOINT ["/blockchain"]
