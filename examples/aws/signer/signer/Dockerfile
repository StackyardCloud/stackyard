FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /signer ./examples/aws/signer/signer
ENTRYPOINT ["/signer"]
