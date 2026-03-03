FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /signer-basic ./examples/aws/signer/signer-basic
ENTRYPOINT ["/signer-basic"]
