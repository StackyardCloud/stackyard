FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /opensearch-advanced ./examples/opensearch/opensearch-advanced
ENTRYPOINT ["/opensearch-advanced"]
