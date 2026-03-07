FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /opensearch ./examples/aws/opensearch
ENTRYPOINT ["/opensearch"]
