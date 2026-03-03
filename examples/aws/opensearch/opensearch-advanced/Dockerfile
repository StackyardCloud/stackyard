FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /opensearch-advanced ./examples/aws/opensearch/opensearch-advanced
ENTRYPOINT ["/opensearch-advanced"]
