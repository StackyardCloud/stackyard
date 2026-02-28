FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /signer-advanced ./examples/signer/signer-advanced
ENTRYPOINT ["/signer-advanced"]
