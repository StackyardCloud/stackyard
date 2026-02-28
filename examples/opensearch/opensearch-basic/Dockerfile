FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /opensearch-basic ./examples/opensearch/opensearch-basic
ENTRYPOINT ["/opensearch-basic"]
